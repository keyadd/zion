package utils

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"strings"
	"zion.com/zion/config"
)

func hasPort(s string) bool {
	// IPv6 address in brackets.
	if strings.LastIndex(s, "[") == 0 {
		return strings.LastIndex(s, ":") > strings.LastIndex(s, "]")
	}

	// Otherwise the presence of a single colon determines if there's a port
	// since IPv6 addresses outside of brackets (count > 1) can't have a
	// port.
	return strings.Count(s, ":") == 1
}

// WsConn 客户端连接模块
func WsConn(config config.Client, v4 string) (*websocket.Conn, error) {
	scheme := "ws"
	if config.TLS == true {
		scheme = "wss"
	}

	encrypt := "0"
	if config.Encrypt == true {
		encrypt = "1"
	}
	u := url.URL{Scheme: scheme, Host: config.Addr, Path: config.Path}
	header := make(http.Header)
	url := u.String() // + "?host=" + url.QueryEscape(config.TunAddr)
	header.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.4844.51 Safari/537.36")
	header.Set("v4", config.V4Addr)
	header.Set("v6", config.V6Addr)
	header.Set("uuid", config.UUID)
	header.Set("encrypt", encrypt)
	fmt.Println(v4)
	fmt.Println(url)

	c, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Printf("[client] failed to dial websocket %v", err)
		return nil, err
	}
	return c, nil
}
