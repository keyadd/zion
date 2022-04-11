package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water/waterutil"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"zion.com/zion/config"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type Server struct {
	wSocket *websocket.Conn // 底层websocket
	mutex   sync.Mutex      // 避免重复关闭管道
	config  config.Server   //配置文件
	cidr    string          //客户端 ip
	encrypt bool            //客户端 ip

}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10000,
	WriteBufferSize: 10000,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var serverConn sync.Map //make(map[string]*websocket.Conn)

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
		go Handler(config, w, r)
	})

	log.Printf("zion ws server start")
	http.ListenAndServe(addr, nil)
}

//#########################################################写连接##########################################################

func Handler(config config.Server, w http.ResponseWriter, r *http.Request) {
	cidr := r.Header.Get("addr")
	encrypt := r.Header.Get("encrypt")
	fmt.Println(encrypt)
	var ch bool = false

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
	go conn.wsToTun(&ch)

	go conn.tunToWs(&ch)

}

func (s *Server) tunToWs(ch *bool) {
	defer func() {
		//serverConn.Delete(s.cidr)
		fmt.Println("退出成功2")

	}()

	buf := make([]byte, 10000)
	for {
		fmt.Println(*ch)
		if *ch == true {
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
		log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", srcIPv4, dstIPv4)

		//加密代码块
		//var encoding []byte
		//fmt.Println(b)
		if s.encrypt == true {
			b = utils.PswEncrypt(b)
		}

		//fmt.Println(b)

		conn, ok := serverConn.Load(dstIPv4)
		if !ok {
			break
		}
		s.mutex.Lock()
		err = conn.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, b)
		s.mutex.Unlock()

		if err != nil {
			log.Println("c.wsSocket.WriteMessage error= ", err)
			break
		}

	}
}

//###################################################################读连接############################################

func (s *Server) wsToTun(ch *bool) {
	defer func() {
		*ch = true
		serverConn.Delete(s.cidr)
		fmt.Println("退出成功1")
	}()
	for {
		load, _ := serverConn.Load(s.cidr)
		_, b, err := (load).(*websocket.Conn).ReadMessage()
		if err != nil || err == io.EOF {
			break
		}

		//解密代码块
		//var decoding []byte
		fmt.Println(b)
		fmt.Println(s.encrypt)
		if s.encrypt == true {
			b = utils.PswDecrypt(b)
		}
		fmt.Println(b)

		//if string(b) == "ping" {
		//	fmt.Println(string(b))
		//	go s.Loop()
		//
		//}

		if !waterutil.IsIPv4(b) {
			continue
		}

		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", srcIPv4, dstIPv4)

		_, err = tunDevice.Write(b)
		if err != nil {
			log.Println("iface.Write error= ", err)
			break
		}

	}
}
