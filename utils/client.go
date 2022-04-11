package utils

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
	"zion.com/zion/config"
)

// ClientConn 客户端连接模块
func ClientConn(config config.Client) (*websocket.Conn, error) {
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
	header.Set("addr", config.TunAddr)
	header.Set("encrypt", encrypt)
	fmt.Println(url)
	fmt.Println(encrypt)
	c, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		log.Printf("[client] failed to dial websocket %v", err)
		return nil, err
	}
	return c, nil
}

func CheckSum(data []byte) uint16 {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += (sum >> 16)

	return uint16(^sum)
}
