package h2

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/tatsushid/go-fastping"
	"golang.org/x/net/http2"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"zion.com/zion/config"
	"zion.com/zion/conn/h2/conn"
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
	h2Socket *conn.Conn         // http2 连接
	mutex    sync.Mutex         // 避免重复关闭管道
	iface    io.ReadWriteCloser //tun 虚拟网卡的接口
	config   config.Client      //全局配置文件
	routes   bool               //是否退出是清空路由配置
	Client   *http.Client

	Method string
	// Header enables sending custom headers to the server
	Header http.Header
}

var gLocker sync.Mutex    //全局锁
var gCondition *sync.Cond //全局条件变量

func (c *Client) Connect(ctx context.Context, config config.Client) (*conn.Conn, *http.Response, error) {

	scheme := "http"
	if config.TLS == true {
		scheme = "https"
	}
	encrypt := "0"
	if config.Encrypt == true {
		encrypt = "1"
	}
	header := make(http.Header)
	//url := u.String() // + "?host=" + url.QueryEscape(config.TunAddr)
	header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36")
	header.Set("addr", config.V4Addr)
	header.Set("encrypt", encrypt)

	reader, writer := io.Pipe()
	u := url.URL{Scheme: scheme, Host: config.Addr, Path: config.Path}
	fmt.Println(u.String())
	c.Method = http.MethodPost
	// Create a request object to send to the server
	req, err := http.NewRequest(c.Method, u.String(), reader)

	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}
	c.Header = header

	// Apply custom headers
	if c.Header != nil {
		req.Header = c.Header
	}
	//req.Close = true

	// Apply given context to the sent request
	req = req.WithContext(ctx)

	// If an http client was not defined, use the default http client
	httpClient := c.Client
	if httpClient == nil {
		httpClient = defaultClient.Client
	}

	// Perform the request
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, nil, err
	}

	// Create a connection
	conn, ctx := conn.NewConn(req.Context(), resp.Body, writer)

	// Apply the connection context on the request context
	resp.Request = req.WithContext(ctx)

	return conn, resp, nil
}

var defaultClient = Client{
	Method: http.MethodPost,
	Client: &http.Client{Transport: &http2.Transport{}},
}

func StartClient(config config.Client, globalBool bool) {

	dnsServers := strings.Split(config.Dns, ",")
	//客户端新建虚拟网卡方法
	tunDev, err := tun.OpenTunDevice(config.Name, config.V4Addr, config.V4Gw, config.V4Mask, dnsServers)
	if err != nil {
		log.Fatalf("failed to open tun device: %v", err)
	}
	//客户端连接服务端方法

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := &Client{
		Client: &http.Client{
			Transport: &http2.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		},
	}

	h2Socket, _, err := d.Connect(ctx, config)
	if err != nil {
		log.Fatalf("Initiate conn: %s", err)
	}
	//defer h2Socket.Close()
	fmt.Println(h2Socket)

	c := &Client{
		h2Socket: h2Socket,
		config:   config,
		iface:    tunDev,
	}
	gLocker.Lock()
	gCondition = sync.NewCond(&gLocker)
	go c.tunToWs()
	go c.wsToTun()

	go c.LoopPing()
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
		//c.h2Socket.ReadCloser.Close()
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
		_, err = c.h2Socket.Write(b)
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
		//c.h2Socket.WriteCloser.Close()
		gCondition.Signal()

	}()
	b := make([]byte, 10000)
	for {

		//c.mutex.Lock()
		_, err := c.h2Socket.Read(b)
		if err != nil {
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
