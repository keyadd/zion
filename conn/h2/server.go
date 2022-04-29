package h2

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"golang.org/x/net/http2"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"zion.com/zion/config"
	"zion.com/zion/conn/h2/conn"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type Server struct {
	mutex      sync.Mutex    // 避免重复关闭管道
	config     config.Server //配置文件
	cidr       string        //客户端 ip
	encrypt    bool          //客户端 ip
	tunDevice  io.ReadWriteCloser
	StatusCode int
}

var serverConn sync.Map

// StartServer 启动方法

var ErrHTTP2NotSupported = fmt.Errorf("HTTP2 not supported")

var h2Upgrader = Server{
	StatusCode: http.StatusOK,
}

func (u *Server) Accept(w http.ResponseWriter, r *http.Request) (*conn.Conn, error) {
	if !r.ProtoAtLeast(2, 0) {
		return nil, ErrHTTP2NotSupported
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrHTTP2NotSupported
	}

	c, ctx := conn.NewConn(r.Context(), r.Body, &flushWrite{w: w, f: flusher})

	// Update the request context with the connection context.
	// If the connection is closed by the server, it will also notify everything that waits on the request context.
	*r = *r.WithContext(ctx)

	w.WriteHeader(u.StatusCode)
	flusher.Flush()

	return c, nil
}

type flushWrite struct {
	w io.Writer
	f http.Flusher
}

func (w *flushWrite) Write(data []byte) (int, error) {
	n, err := w.w.Write(data)
	w.f.Flush()
	return n, err
}

func (w *flushWrite) Close() error {
	// Currently server side close of connection is not supported in Go.
	// The server closes the connection when the http.Handler function returns.
	// We use connection context and cancel function as a work-around.
	return nil
}

func setupCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	(*w).Header().Set("Access-Control-Allow-Headers", "*")
}
func StartServer(config config.Server) {

	dnsServers := strings.Split(config.Dns, ",")

	//开启虚拟网卡方法
	tunDev, err := tun.OpenTunDevice(config.Name, config.V4Addr, config.V4Gw, config.V4Mask, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	var srv http.Server
	//http2.VerboseLogs = true
	srv.Addr = config.Port
	http.HandleFunc(config.Path, func(w http.ResponseWriter, r *http.Request) {
		setupCORS(&w)
		if r.Method == http.MethodPost {
			Handler(tunDev, config, w, r)
		} else {
			io.WriteString(w, "Hello，世界！")
		}
	})
	http2.ConfigureServer(&srv, &http2.Server{})
	go func() {
		log.Fatal(srv.ListenAndServeTLS(config.CertFile, config.KeyFile))
	}()
	select {}
	//mux := http.NewServeMux()
	//mux.HandleFunc(config.Path, func(w http.ResponseWriter, r *http.Request) {
	//	if r.Method == http.MethodPost {
	//		Handler(tunDev, config, w, r)
	//	} else {
	//		io.WriteString(w, "Hello，世界！")
	//	}
	//})
	//
	//h2s := &http2.Server{}
	//h1s := &http.Server{Addr: addr, Handler: h2c.NewHandler(mux, h2s)}
	//log.Printf("zion ws server start")
	//
	//log.Fatal(h1s.ListenAndServe())

}

//#########################################################写连接##########################################################

func Handler(tunDevice io.ReadWriteCloser, config config.Server, w http.ResponseWriter, r *http.Request) {
	cidr := r.Header.Get("addr")
	encrypt := r.Header.Get("encrypt")
	conn, err := h2Upgrader.Accept(w, r)
	if err != nil {
		log.Printf("Failed creating connection from %s: %s", r.RemoteAddr, err)
		return
	}

	fmt.Println(cidr)

	s := &Server{
		config:    config,
		cidr:      cidr,
		tunDevice: tunDevice,
	}
	if encrypt == "1" {
		s.encrypt = true
	} else {
		s.encrypt = false
	}
	serverConn.Store(cidr, conn)
	go s.wsToTun()

	go s.tunToWs()
	select {}

}

func (s *Server) tunToWs() {
	defer func() {
		fmt.Println("exit tunToWs")
	}()
	buf := make([]byte, 10000)
	for {

		_, ok := serverConn.Load(s.cidr)
		if !ok {
			fmt.Println(22)
			break
		}
		n, err := s.tunDevice.Read(buf)
		if err != nil || err == io.EOF || n == 0 {
			continue
		}
		b := buf[:n]

		src, dst := utils.GetIP(b)
		if src == "" || dst == "" {
			continue
		}
		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", srcIPv4, dstIPv4)

		//加密代码块
		if s.encrypt == true {
			b = utils.EncryptChacha1305(b, s.config.Key)
		}

		c, ok := serverConn.Load(dst)
		if !ok {
			continue
		}
		s.mutex.Lock()
		_, err = c.(*conn.Conn).Write(b)
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
	buf := make([]byte, 10000)
	for {
		fmt.Println(111)
		load, _ := serverConn.Load(s.cidr)
		_, err := (load).(*conn.Conn).Read(buf)
		if err != nil {
			fmt.Println(err)
			break
		}

		//解密代码块
		if s.encrypt == true {
			buf = utils.DecryptChacha1305(buf, s.config.Key)
		}

		src, dst := utils.GetIP(buf)
		if src == "" || dst == "" {
			continue
		}

		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", src, dst)

		_, err = s.tunDevice.Write(buf)
		if err != nil {
			log.Println("iface.Write error= ", err)
			break
		}

	}
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
