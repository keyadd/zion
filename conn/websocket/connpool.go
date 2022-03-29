package websocket

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
)

type Pool struct {
	//用于维护用户的map
	clients map[*Server]bool
	//维护用户连接
	clientConn map[string]*websocket.Conn

	//用于用户订阅的chan
	register chan *Server

	//用于用户取消订阅的chan
	unregister chan *Server
}

// NewPool 实例化一个调度器
func NewPool() *Pool {
	return &Pool{
		register:   make(chan *Server),
		unregister: make(chan *Server),
		clients:    make(map[*Server]bool),
		clientConn: make(map[string]*websocket.Conn),
	}
}

//开始运行调度器
func (p *Pool) run() {
	for {
		select {
		case client := <-p.register:
			log.Printf("客户端 %s: 订阅\n", client.cidr)
			p.clients[client] = true
			p.clientConn[client.cidr] = client.wsSocket
			fmt.Println(p.clientConn)
		case client := <-p.unregister:
			log.Printf("客户端 %s: 取消订阅\n", client.cidr)
			if _, ok := p.clients[client]; ok {
				delete(p.clientConn, client.cidr)
				delete(p.clients, client)
			}
		}
	}
}
