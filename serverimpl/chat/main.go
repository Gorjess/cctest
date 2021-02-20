package main

import (
	"cloudcadetest/framework/factory"
	"cloudcadetest/framework/module"
	"cloudcadetest/modconf"
	"cloudcadetest/serverimpl/chat/conf"
	"cloudcadetest/serverimpl/chat/game"
	"cloudcadetest/serverimpl/chat/modules/playergate"
	"cloudcadetest/serverimpl/chat/modules/self"
	"fmt"
	"net/http"
	_ "net/http/pprof"
)

var InstallAt string

func main() {
	if InstallAt == "" {
		InstallAt = "./"
	}
	println("chatserver installed at:", InstallAt)

	// start pprof
	go func() {
		fmt.Println(http.ListenAndServe("localhost:1108", nil))
	}()

	s := factory.New(&modconf.ServerConf{
		LenStackBuf:  4096,
		LogLevel:     "release",
		LogPath:      InstallAt + "/log",
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
