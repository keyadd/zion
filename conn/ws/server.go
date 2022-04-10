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
	log.Printf("zion ws server started on %s,TunAddr is %v", addr, config.TunAddr)

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
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := &Server{
		config:  config,
		wSocket: wsConn,
		cidr:    cidr,
	}
	//serverConn[cidr+pathNode] = wsConn
	serverConn.Store(cidr, wsConn)
	go conn.tunToWs()
	go conn.wsToTun()
}

func (s *Server) tunToWs() {
	defer func() {
		s.mutex.Lock()
		s.wSocket.Close()
		s.mutex.Unlock()

		serverConn.Delete(s.cidr)

	}()

	buffer := make([]byte, 10000)
	for {
		n, err := tunDevice.Read(buffer)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buffer[:n]
		if !waterutil.IsIPv4(b) {
			continue
		}
		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", srcIPv4, dstIPv4)

		//加密代码块
		//var encoding []byte
		if s.config.Encrypt == true {
			//b = utils.PswEncrypt(b)
		}

		conn, ok := serverConn.Load(dstIPv4)
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

func (s *Server) Loop() {
	load, _ := serverConn.Load(s.cidr)
	s.mutex.Lock()
	if err := (load).(*websocket.Conn).WriteMessage(websocket.BinaryMessage, []byte("pong")); err != nil {
		fmt.Println("server heartbeat fail")
	}
	s.mutex.Unlock()
}

//###################################################################读连接############################################

func (s *Server) wsToTun() {
	defer func() {

		serverConn.Delete(s.cidr)
		s.mutex.Lock()
		s.wSocket.Close()
		s.mutex.Unlock()
	}()
	for {
		load, _ := serverConn.Load(s.cidr)
		_, b, err := (load).(*websocket.Conn).ReadMessage()
		if err != nil || err == io.EOF {
			break
		}

		//解密代码块
		//var decoding []byte
		if s.config.Encrypt == true {
			//b = utils.PswDecrypt(b)
		}

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
		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", srcIPv4, dstIPv4)

		_, err = tunDevice.Write(b)
		if err != nil {
			log.Println("iface.Write error= ", err)
			break
		}

	}
}
