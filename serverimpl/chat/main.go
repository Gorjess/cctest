package main

import (
	"cloudcadetest/framework/factory"
	"cloudcadetest/framework/factory/platform"
	"cloudcadetest/framework/module"
	"cloudcadetest/modconf"
	"cloudcadetest/serverimpl/chat/conf"
	"cloudcadetest/serverimpl/chat/game"
	"cloudcadetest/serverimpl/chat/game/entity/player"
	"cloudcadetest/serverimpl/chat/mods/self"
)

func main() {
	game.Server = factory.New(&modconf.ServerConf{
		LenStackBuf:  4096,
		LogLevel:     "release",
		LogPath:      platform.GetLogRootPath(),
		LogFileName:  "server",
		LogChanNum:   100000,
		RollSize:     200,
		EnableStdOut: false,
	})

	game.Server.Run([]module.IModule{self.New(player.New)})
}

func init() {
	conf.Load()
}
