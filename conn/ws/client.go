package ws

import (
	"github.com/gorilla/websocket"
	"github.com/songgao/water/waterutil"
	"io"
	"log"
	"strings"
	"sync"
	"zion.com/zion/config"
	"zion.com/zion/route"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type Client struct {
	wSocket *websocket.Conn // 写websocket
	rSocket *websocket.Conn // 读websocket
	mutex   sync.Mutex      // 避免重复关闭管道
	config  config.Client   //全局配置文件
}

var clientConn sync.Map //= make(map[string]*websocket.Conn)
var clientTunDev io.ReadWriteCloser

func StartClient(config config.Client, globalBool bool) {

	dnsServers := strings.Split(config.TunDns, ",")
	//客户端新建虚拟网卡方法
	tunDev, err := tun.OpenTunDevice(config.TunName, config.TunAddr, config.TunGw, config.TunMask, dnsServers)
	clientTunDev = tunDev
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}

	str := "qwertyuiop"
	for i := 0; i < 2; i++ {
		u := str[i]
		go ClientR(config, string(u))
	}

	//客户端连接服务端方法

	str2 := "asdfghjklz"
	for i := 0; i < 2; i++ {
		u := str2[i]
		go ClientW(config, string(u))
	}

	if globalBool == true {
		route.Route(config.TunName, config.TunDns, config.TunGw, config.Addr)
	}
	log.Printf("zion ws client started,TunAddr is %v", config.TunAddr)
	select {}

}

func ClientR(config config.Client, path string) {
	config.Path = config.Path + "/" + "w"
	wsSocket, err := utils.ClientConn(config, path)
	if err != nil {
		return
	}

	conn := &Client{
		rSocket: wsSocket,
		config:  config,
	}
	clientConn.Store(config.TunAddr+path, wsSocket) //[config.TunAddr+path] = wsSocket
	conn.wsToTun(path)
}

//从服务端获取到是否加密的包 然后解密 发送给tun虚拟网卡
func (c *Client) wsToTun(path string) {
	defer func() {
		clientConn.Delete(c.config.TunAddr + path)
		c.rSocket.Close()
	}()
	for {
		//c.mutex.Lock()
		load, _ := clientConn.Load(c.config.TunAddr + path)
		_, b, err := load.(*websocket.Conn).ReadMessage()
		if err != nil || err == io.EOF {
			break
		}

		//解密代码块
		//var decoding []byte
		if c.config.Encrypt == true {
			b = utils.PswDecrypt(b)
		}
		//c.mutex.Unlock()
		if !waterutil.IsIPv4(b) {
			continue
		}

		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", srcIPv4, dstIPv4)
		clientTunDev.Write(b)
	}
}

func ClientW(config config.Client, path string) {
	config.Path = config.Path + "/" + "r"

	wsSocket, err := utils.ClientConn(config, path)
	if err != nil {
		return
	}
	conn := &Client{
		wSocket: wsSocket,
		config:  config,
	}
	clientConn.Store(config.TunAddr+path, wsSocket)
	conn.tunToWs(path)
}

//发送心跳方法
//func (c *Client) procLoop() {
//	// 启动一个goroutine发送心跳
//	defer c.wsSocket.Close()
//	i := 0
//	for {
//		time.Sleep(30 * time.Second)
//		c.mutex.Lock()
//		if err := c.wsSocket.WriteMessage(ws.TextMessage, []byte("ping")); err != nil {
//			fmt.Println("heartbeat fail")
//			i++
//			if i == 5 {
//				break
//			}
//			continue
//		}
//		c.mutex.Unlock()
//	}

//}

//从tun网卡读取到包 根据配置是否加密 发送到服务端
func (c *Client) tunToWs(path string) {
	defer func() {
		clientConn.Delete(c.config.TunAddr + path)
		c.wSocket.Close()
	}()

	packet := make([]byte, 10000)
	for {
		n, err := clientTunDev.Read(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]
		if !waterutil.IsIPv4(b) {
			continue
		}
		//waterutil.IPv4Protocol()

		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", srcIPv4, dstIPv4)
		//fmt.Println("c.config.TunAddr+path : ", c.config.TunAddr+path)
		//加密代码块
		//var encoding []byte
		if c.config.Encrypt == true {
			b = utils.PswEncrypt(b)
		}
		conn, ok := clientConn.Load(c.config.TunAddr + path)
		if !ok {
			continue
		}
		c.mutex.Lock()

		err = conn.(*websocket.Conn).WriteMessage(websocket.TextMessage, b)

		if err != nil {
			log.Println("Conn.wsSocket.WriteMessage : ", err)
			break
		}
		c.mutex.Unlock()
	}
}