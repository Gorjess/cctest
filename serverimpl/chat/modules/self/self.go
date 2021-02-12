package self

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/module"
	"cloudcadetest/framework/rpc"
	"cloudcadetest/serverimpl/chat/game"
)

var Mod = new(mod)

type mod struct {
	*module.ServerMod
}

func (m *mod) OnInit() {
	sm := &module.ServerMod{
		GoLen:              10000,
		TimerDispatcherLen: 10000,
		RPCServer:          rpc.NewServer(10000),
	}
	sm.Init()
	Mod.ServerMod = sm

	game.Init(sm)
}

func (m *mod) OnDestroy() {
	log.Release("chat module destroyed")
}
