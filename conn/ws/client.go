package websocket

import "C"
import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/songgao/water/waterutil"
	"io"
	"log"
	"strings"
	"sync"
	"time"
	"zion.com/zion/config"
	"zion.com/zion/route"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type Client struct {
	wsSocket   *websocket.Conn    // 底层websocket
	mutex      sync.Mutex         // 避免重复关闭管道
	iface      io.ReadWriteCloser //tun 虚拟网卡的接口
	config     config.Client      //全局配置文件
	clientConn map[string]*websocket.Conn
}

func StartClient(config config.Client) {

	dnsServers := strings.Split(config.TunDns, ",")
	//客户端新建虚拟网卡方法
	tunDev, err := tun.OpenTunDevice(config.TunName, config.TunAddr, config.TunGw, config.TunMask, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	//客户端连接服务端方法
	wsSocket, err := utils.ClientConn(config)
	if err != nil {
		return
	}
	conn := &Client{
		wsSocket: wsSocket,
		config:   config,
		iface:    tunDev,
	}

	go conn.procLoop()
	go conn.wsToTun()
	go conn.tunToWs()

	route.Route(config.TunName, config.TunDns, config.TunGw,config.Addr)

	log.Printf("zion ws client started,TunAddr is %v", config.TunAddr)

	select {}

}

//发送心跳方法
func (c *Client) procLoop() {
	// 启动一个goroutine发送心跳
	defer c.wsSocket.Close()
	i := 0
	for {
		time.Sleep(30 * time.Second)
		c.mutex.Lock()
		if err := c.wsSocket.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
			fmt.Println("heartbeat fail")
			i++
			if i == 5 {
				break
			}
			continue
		}
		c.mutex.Unlock()
	}

}

//从tun网卡读取到包 根据配置是否加密 发送到服务端
func (c *Client) tunToWs() {
	defer c.wsSocket.Close()

	packet := make([]byte, 10000)
	for {
		n, err := c.iface.Read(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]
		if !waterutil.IsIPv4(b) {
			continue
		}
		srcIPv4, dstIPv4 := utils.GetIPv4(b)
		if srcIPv4 == "" || dstIPv4 == "" {
			continue
		}
		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", srcIPv4, dstIPv4)
		if c.config.Encrypt {
			b = utils.XOR(b)
		}
		c.mutex.Lock()
		err = c.wsSocket.WriteMessage(websocket.BinaryMessage, b)

		if err != nil {
			log.Println("Conn.wsSocket.WriteMessage : ", err)
			break
		}
		c.mutex.Unlock()

	}
}

//从服务端获取到是否加密的包 然后解密 发送给tun虚拟网卡
func (c *Client) wsToTun() {
	defer c.wsSocket.Close()
	for {
		//c.mutex.Lock()
		_, b, err := c.wsSocket.ReadMessage()
		if err != nil || err == io.EOF {
			break
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
		if string(b) == "pong" {
			fmt.Println(string(b))
		}
		if c.config.Encrypt {
			b = utils.XOR(b)
		}
		c.iface.Write(b)
	}
}
