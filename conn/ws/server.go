package ws

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water/waterutil"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"zion.com/zion/config"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type Server struct {
	mutex   sync.Mutex    // 避免重复关闭管道
	config  config.Server //配置文件
	cidr    string        //客户端 ip
	encrypt bool          //客户端 ip

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10000,
	WriteBufferSize: 10000,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var serverConn sync.Map

var tunDevice io.ReadWriteCloser

//
func StartServer(config config.Server) {

	addr := "0.0.0.0:" + strconv.Itoa(config.Port)
	dnsServers := strings.Split(config.TunDns, ",")

	//开启虚拟网卡方法
	tunDev, err := tun.OpenTunDevice(config.TunName, config.TunAddr, config.TunGw, config.TunMask, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	tunDevice = tunDev

	//写连接
	http.HandleFunc(config.Path, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(config.Path)

		Handler(config, w, r)
	})

	log.Printf("zion ws server start")
	http.ListenAndServe(addr, nil)
}

//#########################################################写连接##########################################################

func Handler(config config.Server, w http.ResponseWriter, r *http.Request) {
	cidr := r.Header.Get("addr")
	encrypt := r.Header.Get("encrypt")

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := &Server{
		config: config,
		cidr:   cidr,
	}
	if encrypt == "1" {
		conn.encrypt = true
	} else {
		conn.encrypt = false
	}
	fmt.Println(conn.encrypt)
	serverConn.Store(cidr, wsConn)
	go conn.wsToTun()

	go conn.tunToWs()

}

// LoopIcmp 发送一个无返回的icmp包,来安排 tunToWs 退出
func (s *Server) LoopIcmp() {
	conn, err := net.Dial("ip4:icmp", s.cidr)
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

func (s *Server) tunToWs() {
	defer func() {
		fmt.Println("exit tunToWs")
	}()

	buf := make([]byte, 10000)
	for {
		_, ok := serverConn.Load(s.cidr)
		if !ok {
			break
		}
		n, err := tunDevice.Read(buf)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buf[:n]
		if !waterutil.IsIPv4(b) {
			continue
		}
		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", srcIPv4, dstIPv4)

		//加密代码块
		if s.encrypt == true {
			b = utils.EncryptChacha1305(b, s.config.Key)
		}

		conn, ok := serverConn.Load(dstIPv4)
		if !ok {
			break
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

//###################################################################读连接############################################

func (s *Server) wsToTun() {
	defer func() {
		serverConn.Delete(s.cidr)
		s.LoopIcmp()

		fmt.Println("exit wsToTun")
	}()
	for {
		load, _ := serverConn.Load(s.cidr)
		_, b, err := (load).(*websocket.Conn).ReadMessage()
		if err != nil || err == io.EOF {

			break
		}

		//解密代码块
		if s.encrypt == true {
			b = utils.DecryptChacha1305(b, s.config.Key)
		}

		if !waterutil.IsIPv4(b) {
			continue
		}

		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", srcIPv4, dstIPv4)

		_, err = tunDevice.Write(b)
		if err != nil {
			log.Println("iface.Write error= ", err)
			break
		}

	}
}
