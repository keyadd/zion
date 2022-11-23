package ws

import (
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

type Client struct {
	wsSocket *websocket.Conn    // 底层websocket
	mutex    sync.Mutex         // 避免重复关闭管道
	iface    io.ReadWriteCloser //tun 虚拟网卡的接口
	config   config.Client      //全局配置文件
	routes   bool               //是否退出是清空路由配置
	ipv4     string
}

var gLocker sync.Mutex    //全局锁
var gCondition *sync.Cond //全局条件变量

func StartClient(config config.Client, globalBool bool) {

	dnsServers := strings.Split(config.Dns, ",")
	v4, ipv4Net, err := net.ParseCIDR(config.V4Addr)
	gw := ipv4Net.IP.To4()
	gw[3]++
	if err != nil {
		return
	}

	_, ipv6Net, _ := net.ParseCIDR(config.V6Addr)
	to6 := ipv6Net.IP.To16()
	v6gw := to6.String() + "1"
	//客户端新建虚拟网卡方法
	tunDev, err := tun.OpenDevice(config.Name, config.V4Addr, config.V6Addr, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	//客户端连接服务端方法
	wsSocket, err := utils.WsConn(config, v4.String())
	if err != nil {
		//gCondition.Signal()
		fmt.Println(err)
		return
	}
	conn := &Client{
		wsSocket: wsSocket,
		config:   config,
		iface:    tunDev,
		ipv4:     gw.String(),
	}
	gLocker.Lock()
	gCondition = sync.NewCond(&gLocker)

	go conn.tunToWs()
	go conn.wsToTun()

	//开启一个icmp包 ping服务器返回数据  来保持连接 当作心跳
	//go conn.procLoop()
	if globalBool == true {

		//开启系统路由 对系统数据的控制
		err = route.Route(config.Name, config.Dns, gw.String(), v6gw, config.Addr)
		if err != nil {
			gCondition.Signal()
		}
	}

	log.Printf("zion ws client started,TunAddr is %v", config.V4Addr)
	//wg.Wait()
	for {
		//条件变量同步等待
		gCondition.Wait()
		if globalBool {
			route.RetractRoute()
		}
		break
	}

}

//发送心跳方法
func (c *Client) procLoop() {
	// 启动一个goroutine发送心跳
	defer c.wsSocket.Close()
	i := 0
	for {

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
		time.Sleep(15 * time.Second)
	}

}

//
//func (c *Client) LoopPing() {
//
//	//num := 0
//
//	for {
//		time.Sleep(15 * time.Second)
//		pinger, err := ping.NewPinger(c.ipv4) //ping的地址，可以是www.baidu.com，也可以是127.0.0.1
//		if err != nil {
//			fmt.Println(err)
//			gCondition.Signal()
//		}
//		pinger.Count = 1              // ping的次数
//		PINGTIME := time.Duration(10) // ping的时间
//		pinger.Timeout = time.Duration(PINGTIME * time.Second)
//		pinger.SetPrivileged(true)
//		pinger.Run() // blocks until finished
//		stats := pinger.Statistics()
//		//if stats.PacketsRecv >= 1 {
//		//
//		//}
//		fmt.Println(stats)
//		if stats.PacketsRecv == 0 {
//			gCondition.Signal()
//			break
//		}
//		//if stats.PacketsRecv == 0 {
//		//	num++
//		//	continue
//		//}
//		//if ip != c.config.V4Gw {
//		//	gCondition.Signal()
//		//	break
//		//}
//		//
//		//if num == 2 {
//		//	fmt.Println(err)
//		//	gCondition.Signal()
//		//	break
//		//}
//	}
//
//}

//从tun网卡读取到包 根据配置是否加密 发送到服务端
func (c *Client) tunToWs() {
	defer func() {
		fmt.Println("exit tunToWs")
		c.wsSocket.Close()
		gCondition.Signal()
		//route.RetractRoute()
		//wg.Done()
	}()

	packet := make([]byte, 100000)
	for {
		n, err := c.iface.Read(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]

		if waterutil.IPv4Protocol(b) == waterutil.UDP {
			port := utils.PeekDestinationPort(b)
			if port == utils.COMMON_DNS_PORT {
				query, s, _ := utils.ParseDNSQuery(b[28:])
				fmt.Println(query)
				fmt.Println(s)

				//if s == "cip.cc" {
				//	conn, err := net.Dial("ip", "192.168.1.79")
				//	if err != nil {
				//		fmt.Println(err.Error())
				//		return
				//	}
				//	defer conn.Close()
				//
				//	if _, err := conn.Write(b); err != nil {
				//		fmt.Println(err.Error())
				//		return
				//	}
				//
				//}

			}
		}
		src, dst := utils.GetIP(b)
		if src == nil || dst == nil {
			continue
		}

		//log.Printf("srcIPv4: %s tunToWs dstIPv4: %s\n", src, dst)
		//加密代码
		if c.config.Encrypt {
			b = utils.EncryptChacha1305(b, c.config.Key)
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
	defer func() {

		fmt.Println("exit wsToTun")
		c.wsSocket.Close()
		gCondition.Signal()

	}()
	for {
		//c.mutex.Lock()
		_, b, err := c.wsSocket.ReadMessage()
		if err != nil || err == io.EOF {
			fmt.Println(err)
			break
		}

		//解密代码
		if c.config.Encrypt {
			b = utils.DecryptChacha1305(b, c.config.Key)
		}
		//c.mutex.Unlock()

		if b == nil {
			continue
		}
		//if !utils.IsIPv4(b) || !utils.IsIPv6(b) {
		//	continue
		//}
		src, dst := utils.GetIP(b)
		if src == nil || dst == nil {
			continue
		}
		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", src, dst)

		c.iface.Write(b)
	}
}
