package network

import (
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPServer struct {
	Addr            string
	FuncMaxConnNum  func() int
	PendingWriteNum int
	NewAgent        func(*TCPConn) agent.Agent
	ln              net.Listener

	offChan             chan bool // 是否退出了
	NumberOfConn        int32     // 本次统计内的连接数
	ConnNumberPerSecond int32     // 1s允许的连接数

	conns      ConnSet
	mutexConns sync.Mutex // 不是很优雅，暂时先这样做
	wgLn       sync.WaitGroup
	wgConns    sync.WaitGroup
}

const RepeatCnt = 3

func (server *TCPServer) Start() {
	err := server.init()
	if err != nil {
		log.Fatal("tcpserver init fail:%s", err.Error())
		return
	}

	if server.ConnNumberPerSecond > 0 { // 做了连接限制
		go server.tick()
	}

	go server.run()
}

func (server *TCPServer) init() error {
	i := 0
	var ln net.Listener
	var err error
	for {
		ln, err = net.Listen("tcp4", server.Addr)
		if err != nil || ln == nil {
			if i < RepeatCnt {
				i++
				time.Sleep(time.Second)
				continue
			}
			return err
		} else {
			//成功
			break
		}
	}

	log.Release("listen tcp[%v]", server.Addr)

	if server.PendingWriteNum <= 0 {
		server.PendingWriteNum = 100
	}
	if server.NewAgent == nil {
		return errors.New("NewAgent must not be nil")
	}

	server.ln = ln
	server.conns = make(ConnSet)
	return nil
}

/*
 * 开协程的目的主要是想借用系统的time来处理 而不愿采用加减时间来判定
 * 开协程只涉及到原子操作，对于开销上可不计
 */
func (server *TCPServer) tick() {
	server.wgLn.Add(1)
	defer server.wgLn.Done()

	server.offChan = make(chan bool)
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-server.offChan:
			return
		case <-t.C:
			atomic.StoreInt32(&server.NumberOfConn, 0)
		}
	}
}

// wrapper for closing net.Conn
func closeConn(conn net.Conn) {
	if conn == nil {
		return
	}
	e := conn.Close()
	if e != nil {
		log.Error("close conn[%s] failed:%s", conn.RemoteAddr(), e.Error())
	}
}

func (server *TCPServer) run() {
	server.wgLn.Add(1)
	defer func() {
		if server.ConnNumberPerSecond > 0 {
			server.offChan <- true
		}
		server.wgLn.Done()
	}()

	var tempDelay time.Duration
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Release("accept error:%s; retrying in %s", err.Error(), tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		tempDelay = 0

		if server.ConnNumberPerSecond > 0 { // 开启了限流
			num := atomic.AddInt32(&server.NumberOfConn, 1)
			if num >= server.ConnNumberPerSecond { // 超过每秒允许的连接数 直接断开
				closeConn(conn)
				log.Warn("too many connections per second [%s->%s]", num, server.ConnNumberPerSecond)
				continue
			}
		}

		maxConnNum := 0
		if server.FuncMaxConnNum != nil {
			maxConnNum = server.FuncMaxConnNum()
		}
		if maxConnNum == 0 {
			maxConnNum = 100
		}

		server.mutexConns.Lock()
		if len(server.conns) >= maxConnNum {
			server.mutexConns.Unlock()
			closeConn(conn)
			log.Release("too many connections %v", maxConnNum)
			continue
		}

		server.conns[conn] = struct{}{}
		server.mutexConns.Unlock()

		server.wgConns.Add(1)

		tcpConn := newTCPConn(conn, server.PendingWriteNum)
		go server.newAgent(tcpConn)
	}
}

func (server *TCPServer) Close() {
	e := server.ln.Close()
	if e != nil {
		log.Error("close server listener failed:%s", e.Error())
	}
	server.wgLn.Wait()

	server.mutexConns.Lock()
	for conn := range server.conns {
		closeConn(conn)
	}
	server.conns = nil
	server.mutexConns.Unlock()
	server.wgConns.Wait()
}

func (server *TCPServer) newAgent(tcpConn *TCPConn) {
	server.NewAgent(tcpConn)

	// cleanup
	server.mutexConns.Lock()
	delete(server.conns, tcpConn.conn)
	server.mutexConns.Unlock()
	server.wgConns.Done()
}
