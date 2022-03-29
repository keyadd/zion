package websocket

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
	Pool     *Pool
	wsSocket *websocket.Conn    // 底层websocket
	iface    io.ReadWriteCloser //*water.Interface //网关接口
	mutex    sync.Mutex         // 避免重复关闭管道
	config   config.Server      //配置文件
	cidr     string             //客户端 ip
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  10000,
	WriteBufferSize: 10000,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartServer(config config.Server) {
	pool := NewPool()
	go pool.run()

	addr := "0.0.0.0:" + strconv.Itoa(config.Port)
	dnsServers := strings.Split(config.TunDns, ",")
	tunDev, err := tun.OpenTunDevice(config.TunName, config.TunAddr, config.TunGw, config.TunMask, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	log.Printf("zion ws server started on %s,TunAddr is %v", addr, config.TunAddr)
	http.HandleFunc(config.Path, func(w http.ResponseWriter, r *http.Request) {
		wsHandler(pool, tunDev, config, w, r)
	})
	http.ListenAndServe(addr, nil)
}

func wsHandler(pool *Pool, iface io.ReadWriteCloser, config config.Server, w http.ResponseWriter, r *http.Request) {
	cidr := r.Header.Get("addr")
	fmt.Println(cidr)
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn := &Server{
		iface:    iface,
		config:   config,
		Pool:     pool,
		wsSocket: wsConn,
		cidr:     cidr,
	}

	pool.register <- conn //将用户连接注册到 连接池
	go conn.wsToTun()
	go conn.tunToWs()

	log.Printf("zion ws server start")

}

func (s *Server) tunToWs() {
	defer func() {
		s.mutex.Lock()
		s.wsSocket.Close()
		s.mutex.Unlock()
		s.Pool.unregister <- s

	}()

	buffer := make([]byte, 10000)
	for {

		n, err := s.iface.Read(buffer)
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
		if s.config.Encrypt {
			b = utils.XOR(b)
		}
		conn, ok := s.Pool.clientConn[dstIPv4]
		if !ok {
			continue
		}
		s.mutex.Lock()
		err = conn.WriteMessage(websocket.BinaryMessage, b)

		if err != nil {
			log.Println("c.wsSocket.WriteMessage error= ", err)
			break
		}
		s.mutex.Unlock()

	}
}

func (s *Server) wsToTun() {
	defer func() {
		s.Pool.unregister <- s
		s.wsSocket.Close()
	}()
	for {

		_, b, err := s.wsSocket.ReadMessage()
		if err != nil || err == io.EOF {
			break
		}
		if string(b) == "ping" {
			fmt.Println(string(b))
			if err := s.wsSocket.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
				fmt.Println("server heartbeat fail")
				break
			}
		}
		if s.config.Encrypt {
			b = utils.XOR(b)
		}
		if !waterutil.IsIPv4(b) {
			continue
		}

		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", srcIPv4, dstIPv4)
		_, err = s.iface.Write(b)
		if err != nil {
			log.Println("iface.Write error= ", err)
			break
		}

	}
}
