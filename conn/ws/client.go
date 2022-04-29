package ws

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tatsushid/go-fastping"
	"io"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"
	"zion.com/zion/config"
	"zion.com/zion/route"
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

type Client struct {
	wsSocket *websocket.Conn    // 底层websocket
	mutex    sync.Mutex         // 避免重复关闭管道
	iface    io.ReadWriteCloser //tun 虚拟网卡的接口
	config   config.Client      //全局配置文件
	routes   bool               //是否退出是清空路由配置
}

var gLocker sync.Mutex    //全局锁
var gCondition *sync.Cond //全局条件变量

func StartClient(config config.Client, globalBool bool) {

	runtime.GOMAXPROCS(2)

	dnsServers := strings.Split(config.Dns, ",")
	//客户端新建虚拟网卡方法
	tunDev, err := tun.OpenTunDevice(config.Name, config.V4Addr, config.V4Gw, config.V4Mask, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	//客户端连接服务端方法
	wsSocket, err := utils.WsConn(config)
	if err != nil {
		//gCondition.Signal()
		fmt.Println(err)
		return
	}
	conn := &Client{
		wsSocket: wsSocket,
		config:   config,
		iface:    tunDev,
	}
	gLocker.Lock()
	gCondition = sync.NewCond(&gLocker)
	go conn.tunToWs()
	go conn.wsToTun()

	go conn.LoopPing()
	if globalBool == true {
		err = route.Route(config.Name, config.Dns, config.V4Gw, config.Addr)
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

func (c *Client) LoopPing() {
	p := fastping.NewPinger()
	ra, err := net.ResolveIPAddr("ip4:icmp", c.config.V4Gw)
	if err != nil {
		fmt.Println(err)
		gCondition.Signal()
	}
	var ip string
	for {
		time.Sleep(20 * time.Second)
		p.AddIPAddr(ra)
		p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
			fmt.Printf("IP Addr: %s receive, RTT: %v\n", addr.String(), rtt)
			ip = addr.String()
		}

		p.OnIdle = func() {
			return
		}
		err = p.Run()
		if err != nil {
			fmt.Println(err)
			gCondition.Signal()
			break
		}
		if ip != c.config.V4Gw {
			gCondition.Signal()
			break
		}

	}

}

//从tun网卡读取到包 根据配置是否加密 发送到服务端
func (c *Client) tunToWs() {
	defer func() {
		fmt.Println("exit tunToWs")
		c.wsSocket.Close()
		gCondition.Signal()
		//route.RetractRoute()
		//wg.Done()
	}()

	packet := make([]byte, 10000)
	for {
		n, err := c.iface.Read(packet)
		if err != nil || n == 0 {
			continue
		}
		b := packet[:n]

		src, dst := utils.GetIP(b)
		if src == "" || dst == "" {
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
		if src == "" || dst == "" {
			continue
		}
		//log.Printf("srcIPv4: %s wsToTun dstIPv4: %s\n", src, dst)

		c.iface.Write(b)
	}
}
