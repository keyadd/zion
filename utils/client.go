package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"golang.org/x/net/http2"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"zion.com/zion/config"
)

type HttpClientStream struct {
	io.ReadCloser
	io.WriteCloser
}

func newTransport() (ts http.RoundTripper) {
	// TODO Need a timing to refresh the cached server address (if the DNS results changed)
	var (
		mu           sync.Mutex
		resolvedAddr net.Addr
	)
	dial := func(network, addr string) (conn net.Conn, err error) {
		mu.Lock()
		locked := true
		if resolvedAddr != nil {
			addr = resolvedAddr.String()
			mu.Unlock()
			locked = false
		}
		defer func() {
			if locked {
				mu.Unlock()
			}
		}()

		//dial := func(network, addr string) (net.Conn, error)
		//if dial == nil {
		dial := net.Dial
		//}
		conn, err = dial(network, addr)
		if err != nil {
			return nil, err
		}

		if locked {
			resolvedAddr = conn.RemoteAddr()
			mu.Unlock()
			locked = false
		}

		return conn, err
	}

	return &http2.Transport{
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			conn, err := dial(network, addr)
			if err != nil {
				return nil, err
			}
			//if !c.UseH2C {
			//	cn := tls.Client(conn, cfg)
			//	if err := cn.Handshake(); err != nil {
			//		return nil, err
			//	}
			//	if !cfg.InsecureSkipVerify {
			//		if err := cn.VerifyHostname(cfg.ServerName); err != nil {
			//			return nil, err
			//		}
			//	}
			//	state := cn.ConnectionState()
			//	if p := state.NegotiatedProtocol; p != http2.NextProtoTLS {
			//		return nil, fmt.Errorf("http2: unexpected ALPN protocol %q; want %q", p, http2.NextProtoTLS)
			//	}
			//	if !state.NegotiatedProtocolIsMutual {
			//		return nil, errors.New("http2: could not negotiate protocol mutually")
			//	}
			//	conn = cn
			//}
			return conn, nil
		},
		//TLSClientConfig: *tls.Config,
	}
}

// H2Conn 客户端连接模块
func doReq(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		// Skip TLS dial
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println(" client.Do(req)", err)
	}

	return res, nil
}

func H2Conn(config config.Client) (c *HttpClientStream) {
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
	u := url.URL{Scheme: scheme, Host: config.Addr, Path: config.Path}
	fmt.Println(u.String())
	r, w := io.Pipe()

	req := &http.Request{
		URL:    &u,
		Header: header,
		Body:   ioutil.NopCloser(r),
	}
	fmt.Println(req)

	res, err := doReq(req.WithContext(context.TODO()))

	//resp, err := client.Get(url)
	if err != nil {
		log.Fatal(fmt.Errorf("get response error: %v", err))
		return nil
	}

	h2Socket := &HttpClientStream{res.Body, w}
	fmt.Println(h2Socket)

	//resp.Header = header
	return h2Socket
}

// WsConn 客户端连接模块
func WsConn(config config.Client) (*websocket.Conn, error) {
	scheme := "ws"
	if config.TLS == true {
		scheme = "wss"
	}

	encrypt := "0"
	if config.Encrypt == true {
		encrypt = "1"
	}
	//d := websocket.Dialer{
	//	ReadBufferSize:  10000,
	//	WriteBufferSize: 10000,
	//
	//}

	u := url.URL{Scheme: scheme, Host: config.Addr, Path: config.Path}
	header := make(http.Header)
	url := u.String() // + "?host=" + url.QueryEscape(config.TunAddr)
	header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36")
	header.Set("addr", config.V4Addr)
	header.Set("encrypt", encrypt)
	fmt.Println(url)
	c, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Printf("[client] failed to dial websocket %v", err)
		return nil, err
	}
	return c, nil
}
