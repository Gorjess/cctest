package main

import (
	"cloudcadetest/framework/factory"
	"cloudcadetest/framework/factory/platform"
	"cloudcadetest/framework/module"
	"cloudcadetest/modconf"
	"cloudcadetest/serverimpl/chat/conf"
	"cloudcadetest/serverimpl/chat/game"
	"cloudcadetest/serverimpl/chat/modules/playergate"
	"cloudcadetest/serverimpl/chat/modules/self"
)

func main() {
	s := factory.New(&modconf.ServerConf{
		LenStackBuf:  4096,
		LogLevel:     "release",
		LogPath:      platform.GetLogRootPath(),
		LogFileName:  "server",
		LogChanNum:   100000,
		RollSize:     200,
		EnableStdOut: false,
	})

	s.Run([]module.IModule{
		playergate.New(game.NewPlayer),
		self.Mod,
	})
}

func init() {
	conf.Load()
}
