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
	"strings"
	"sync"
	"time"
	"zion.com/zion/config"
	"zion.com/zion/route"
	"zion.com/zion/tun"
	"zion.com/zion/utils"
)

type ICMP struct {
	Type        uint8
	Code        uint8
	Checksum    uint16
	Identifier  uint16
	SequenceNum uint16
}

type Client struct {
	wsSocket *websocket.Conn    // 底层websocket
	mutex    sync.Mutex         // 避免重复关闭管道
	iface    io.ReadWriteCloser //tun 虚拟网卡的接口
	config   config.Client      //全局配置文件
}

func StartClient(config config.Client, globalBool bool) {

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

	go conn.wsToTun()
	go conn.tunToWs()
	go conn.LoopIcmp()

	if globalBool == true {
		route.Route(config.TunName, config.TunDns, config.TunGw, config.Addr)
	}

	log.Printf("zion ws client started,TunAddr is %v", config.TunAddr)

	select {}

}

func (c *Client) LoopIcmp() {
	defer c.wsSocket.Close()
	var (
		icmp  ICMP
		laddr net.IPAddr = net.IPAddr{IP: net.ParseIP(c.config.TunAddr)} //***IP地址改成你自己的网段***
		raddr net.IPAddr = net.IPAddr{IP: net.ParseIP(c.config.TunGw)}
	)
	conn, err := net.DialIP("ip4:icmp", &laddr, &raddr)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer conn.Close()
	icmp.Type = 8 //8->echo message  0->reply message
	icmp.Code = 0
	icmp.Checksum = 0
	icmp.Identifier = 0
	icmp.SequenceNum = 0

	var (
		buffer bytes.Buffer
	)
	//先在buffer中写入icmp数据报求去校验和
	binary.Write(&buffer, binary.BigEndian, icmp)
	icmp.Checksum = utils.CheckSum(buffer.Bytes())
	//然后清空buffer并把求完校验和的icmp数据报写入其中准备发送
	buffer.Reset()
	binary.Write(&buffer, binary.BigEndian, icmp)

	for {
		time.Sleep(30 * time.Second)
		if _, err := conn.Write(buffer.Bytes()); err != nil {
			fmt.Println(err.Error())
			break
		}
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
			//b = utils.XOR(b)
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

		if string(b) == "pong" {
			fmt.Println(string(b))
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

		if c.config.Encrypt {
			//b = utils.XOR(b)
		}
		c.iface.Write(b)
	}
}
