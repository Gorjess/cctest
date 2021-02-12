package playergate

import (
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/network"
	"cloudcadetest/serverimpl/chat/conf"
)

type NewAgentFunc func(*network.TCPConn) agent.Agent

type Gate struct {
	TCPAddr          string
	FuncMaxConnNum   func() int
	PendingWriteNum  int
	ConnNumPerSecond int32 // 每秒限定的连接数
	NewAgent         NewAgentFunc
	tcpServer        *network.TCPServer
}

func New(newAgent NewAgentFunc) *Gate {
	return &Gate{
		TCPAddr:          "0.0.0.0:3066",
		FuncMaxConnNum:   func() int { return conf.Server.MaxConnNum },
		PendingWriteNum:  conf.Server.GatePendingWriteNum,
		ConnNumPerSecond: conf.Server.ConnNumPerSecond,
		NewAgent:         newAgent,
	}
}

func (gate *Gate) Run(closeSig chan bool) {
	log.Release("starting gate module %s", gate.TCPAddr)
	if gate.TCPAddr != "" {
		gate.tcpServer = new(network.TCPServer)
		gate.tcpServer.Addr = gate.TCPAddr
		gate.tcpServer.FuncMaxConnNum = gate.FuncMaxConnNum
		gate.tcpServer.PendingWriteNum = gate.PendingWriteNum
		gate.tcpServer.NewAgent = gate.NewAgent
		gate.tcpServer.ConnNumberPerSecond = gate.ConnNumPerSecond
		gate.tcpServer.Start()
	}

	<-closeSig
}

func (gate *Gate) OnInit() {

}

func (gate *Gate) OnDestroy() {
	if gate.tcpServer != nil {
		gate.tcpServer.Close()
	}
	log.Release("gate destroyed")
}

func (gate *Gate) CloseTCPServer() {
	if gate.tcpServer != nil {
		gate.tcpServer.Close()
	}
}
