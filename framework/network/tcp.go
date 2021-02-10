package network

import (
	"cloudcadetest/framework/log"
	"errors"
	"io"
	"net"
	"time"
)

const lingerSecs = 5

type ConnSet map[net.Conn]struct{}

type TCPConn struct {
	conn      net.Conn
	writeChan chan []byte
	isClosed  bool
}

func newTCPConn(conn net.Conn, pendingWriteNum int) *TCPConn {
	tcpConn := new(TCPConn)
	tcpConn.conn = conn
	tcpConn.writeChan = make(chan []byte, pendingWriteNum)
	return tcpConn
}

func (tcpConn *TCPConn) Destroy() {
	e := tcpConn.conn.(*net.TCPConn).SetLinger(lingerSecs)
	if e != nil {
		log.Error("set linger seconds failed:%s", e.Error())
	}

	time.Sleep(time.Second * 3)

	log.Debug("do close conn:%s", tcpConn.conn.RemoteAddr().String())
	e = tcpConn.conn.Close()
	if e != nil {
		log.Error("close tcp conn failed:%s", e.Error())
	}
}

func (tcpConn *TCPConn) Close() {
	if tcpConn.isClosed {
		return
	}
	close(tcpConn.writeChan)
	tcpConn.isClosed = true
}

func (tcpConn *TCPConn) Write(b []byte) error {
	if b == nil {
		return nil
	}

	if tcpConn.isClosed {
		return errors.New("closed write channel")
	}

	select {
	case tcpConn.writeChan <- b:
	default:
		return errors.New("full write channel")
	}

	return nil
}

func (tcpConn *TCPConn) Read(b []byte) (int, error) {
	return tcpConn.conn.Read(b)
}

func (tcpConn *TCPConn) ReadFull(b []byte) error {
	_, err := io.ReadFull(tcpConn.conn, b)
	return err
}

func (tcpConn *TCPConn) LocalAddr() net.Addr {
	return tcpConn.conn.LocalAddr()
}

func (tcpConn *TCPConn) RemoteAddr() net.Addr {
	return tcpConn.conn.RemoteAddr()
}

func (tcpConn *TCPConn) WriteTask() {
	for b := range tcpConn.writeChan {
		n, err := tcpConn.conn.Write(b)
		if err != nil {
			log.Warn("tcpconn write fail[%s]", err.Error())
			continue
		}

		log.Release("tcp write to %s [%d-%d]", tcpConn.conn.RemoteAddr(), len(b), n)
	}

	tcpConn.Destroy()
}
