package ws

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"zion.com/zion/config"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type Icmp struct {
	Type        uint8
	Code        uint8
	Checksum    uint16
	Identifier  uint16
	SequenceNum uint16
}

type Server struct {
	mutex     sync.Mutex         //避免重复多线程读一份数据 并行转换串行
	conf      config.Server      //配置文件
	uuid      string             //客户端 uuid
	v4        string             //ipv4客户端地址
	v6        string             //ipv6客户端地址
	encrypt   bool               //是否加密
	keys      string             //加密密钥 非对称加密
	tunDevice io.ReadWriteCloser //虚拟网卡
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  100000,
	WriteBufferSize: 100000,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var serverConn sync.Map //存放所以客户端的连接
var userInfo sync.Map   //存放所有客户端的配置文件  （v4  v6  uuid  key）

// StartServer 启动方法
func StartServer(config config.Server) {

	dnsServers := strings.Split(config.Dns, ",")
	for k, a := range config.User {
		userInfo.Store(k, *a)
	}

	//开启虚拟网卡方法
	tunDev, err := tun.OpenDevice(config.Name, config.V4Addr, config.V6Addr, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}

	http.HandleFunc(config.Path, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(config.Path)
		Handler(tunDev, config, w, r)
	})

	log.Printf("zion ws server start")
	http.ListenAndServe(config.Port, nil)
}

//#########################################################写连接##########################################################

func Handler(tunDevice io.ReadWriteCloser, conf config.Server, w http.ResponseWriter, r *http.Request) {
	v4 := r.Header.Get("v4")
	v6 := r.Header.Get("v6")
	uuid := r.Header.Get("uuid")
	encrypt := r.Header.Get("encrypt")

	ipv4, _, err := net.ParseCIDR(v4)
	if err != nil {
		fmt.Println("v4 error", err)
	}

	ipv6, _, err := net.ParseCIDR(v6)
	if err != nil {
		fmt.Println("v4 error", err)
	}

	var keys = ""

	userInfo.Range(func(key, value interface{}) bool {
		if v4 == value.(config.User).V4 && v6 == value.(config.User).V6 && uuid == value.(config.User).UUID {
			keys = value.(config.User).Key
			return true
		}
		return true
	})
	if keys == "" {
		return
	}
	fmt.Println(keys)

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := &Server{
		conf:      conf,
		uuid:      uuid,
		v4:        ipv4.String(),
		v6:        ipv6.String(),
		tunDevice: tunDevice,
		keys:      keys,
	}
	if encrypt == "1" {
		conn.encrypt = true
	} else {
		conn.encrypt = false
	}
	fmt.Println(conn.uuid)
	serverConn.Store(uuid, wsConn)
	go conn.wsToTun()

	go conn.tunToWs()

}

func (s *Server) tunToWs() {
	defer func() {
		fmt.Println("exit tunToWs")
	}()

	buf := make([]byte, 100000)
	for {
		_, ok := serverConn.Load(s.uuid)
		if !ok {
			break
		}
		n, err := s.tunDevice.Read(buf)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buf[:n]

		src, dst := utils.GetIP(b)
		if src == nil || dst == nil {
			continue
		}
		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", src, dst)

		//加密代码块
		if s.encrypt == true {
			b = utils.EncryptChacha1305(b, s.keys)
		}

		if string(b) == "ping" {
			conn, ok := serverConn.Load(s.uuid)
			if !ok {
				continue
			}
			fmt.Println(string(b))
			if err := conn.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, []byte("pong")); err != nil {
				fmt.Println("server heartbeat fail")
				break
			}
		}
		//根据数据包的ip，判断是否是ipv4 还是ipv6 根据ip判断转发到那一个客户端ip上
		if dst.To4() != nil {
			// is ipv4

			if dst.String() == s.v4 {

				conn, ok := serverConn.Load(s.uuid)
				if !ok {
					continue
				}
				s.mutex.Lock()
				err = conn.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, b)

				if err != nil {
					log.Println("c.wsSocket.WriteMessage error= ", err)
					break
				}
				s.mutex.Unlock()
			} else {
				uuid := ""
				userInfo.Range(func(key, value interface{}) bool {
					if dst.String() == value.(config.User).V4 {
						uuid = value.(config.User).UUID
					}
					return true
				})
				if uuid == "" {
					continue
				}
				conn, ok := serverConn.Load(uuid)
				if !ok {
					continue
				}
				s.mutex.Lock()
				err = conn.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, b)

				if err != nil {
					log.Println("c.wsSocket.WriteMessage error= ", err)
					break
				}
				s.mutex.Unlock()

			}
		} else {
			//is ipv6

			if dst.String() == s.v6 {

				conn, ok := serverConn.Load(s.uuid)
				if !ok {
					continue
				}
				s.mutex.Lock()
				err = conn.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, b)

				if err != nil {
					log.Println("c.wsSocket.WriteMessage error= ", err)
					break
				}
				s.mutex.Unlock()
			} else {
				uuid := ""
				userInfo.Range(func(key, value interface{}) bool {
					if dst.String() == value.(config.User).V6 {
						uuid = value.(config.User).UUID
					}
					return true
				})
				if uuid == "" {
					continue
				}
				conn, ok := serverConn.Load(uuid)
				if !ok {
					continue
				}
				s.mutex.Lock()
				err = conn.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, b)

				if err != nil {
					log.Println("c.wsSocket.WriteMessage error= ", err)
					break
				}
				s.mutex.Unlock()

			}
		}
	}
}

//###################################################################读连接############################################

func (s *Server) wsToTun() {
	defer func() {
		serverConn.Delete(s.uuid)
		s.LoopIcmp()

		fmt.Println("exit wsToTun")
	}()
	for {
		load, ok := serverConn.Load(s.uuid)
		if !ok {
			break
		}
		_, b, err := (load).(*websocket.Conn).ReadMessage()
		if err != nil || err == io.EOF {
			break
		}

		//解密代码块
		if s.encrypt == true {
			b = utils.DecryptChacha1305(b, s.keys)
		}

		src, dst := utils.GetIP(b)
		if src == nil || dst == nil {
			continue
		}

		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", src, dst)

		_, err = s.tunDevice.Write(b)
		if err != nil {
			log.Println("iface.Write error= ", err)
			break
		}

	}
}

// LoopIcmp 发送一个无返回的icmp包,来安排 tunToWs 退出
func (s *Server) LoopIcmp() {
	conn, err := net.Dial("ip4:icmp", s.v4)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer conn.Close()
	icmp := Icmp{
		Type:        8,
		Code:        0,
		Checksum:    0,
		Identifier:  0,
		SequenceNum: 0,
	}
	var (
		buffer bytes.Buffer
	)
	//发送一个单独的 icmp包 ,无icmp返回包
	binary.Write(&buffer, binary.BigEndian, icmp)

	if _, err := conn.Write(buffer.Bytes()); err != nil {
		fmt.Println(err.Error())
		return
	}
	return

}
