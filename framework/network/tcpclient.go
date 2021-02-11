package network

import (
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"net"
	"sync"
	"time"
)

type TCPClient struct {
	Addr            string
	ConnectInterval time.Duration
	PendingWriteNum int
	NewAgent        func(*TCPConn) agent.Agent
	conn            net.Conn
	closeFlag       bool
	ReconnectFlag   bool //能否断线重连
	DisconnectCB    func(string)
	sync.Mutex
}

func (client *TCPClient) Start() {
	client.init()
	client.run()
}

func (client *TCPClient) init() {
	if client.ConnectInterval <= 0 {
		client.ConnectInterval = 3 * time.Second
		log.Release("invalid ConnectInterval, reset to %v", client.ConnectInterval)
	}
	if client.PendingWriteNum <= 0 {
		client.PendingWriteNum = 100
		log.Release("invalid PendingWriteNum, reset to %v", client.PendingWriteNum)
	}
	if client.NewAgent == nil {
		log.Fatal("NewAgent must not be nil")
	}
	if client.conn != nil {
		log.Fatal("client is running")
	}

	client.closeFlag = false
	client.ReconnectFlag = true
}

func (client *TCPClient) run() {
	for {
		//不能断线重连
		if !client.ReconnectFlag {
			break
		}
		if client.closeFlag {
			//服务器关闭
			break
		}

		if !client.connect() { //connect返回代表连接失败或中途断开连接，在该for语句都会再次重连
			continue
		}
	}
	if client.DisconnectCB != nil {
		client.DisconnectCB(client.Addr)
	}
}

func (client *TCPClient) connect() bool {
	c := client.dial()
	if c == nil {
		return false
	}

	if client.closeFlag {
		c.Close()
		return false
	}

	client.Lock()
	client.conn = c
	client.Unlock()

	conn := newTCPConn(client.conn, client.PendingWriteNum)
	client.NewAgent(conn) //回调逻辑里直接读数据，当读取失败，NewAgent才返回，上层逻辑会继续调用该connect函数(断线重连)

	client.Lock()
	defer client.Unlock()

	client.conn = nil

	return true
}

func (client *TCPClient) dial() net.Conn {
	conn, err := net.Dial("tcp", client.Addr)
	if err == nil {
		//连接成功
		tcpConn, ok := conn.(*net.TCPConn)
		if ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(time.Second * 5)
		}
		return conn
	}

	log.Warn("connect to %v error: %v", client.Addr, err)
	time.Sleep(client.ConnectInterval)
	return nil
}

func (client *TCPClient) Close() {
	client.Lock()
	defer client.Unlock()

	if client.closeFlag {
		return
	}

	if client.conn != nil {
		client.conn.Close()
		client.conn = nil
	}

	client.closeFlag = true
}
