package network

import "net"

type IConn interface {
	Read(data []byte) (int, error)
	ReadFull(data []byte) error
	Write(args []byte) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()

	WriteTask()
}
