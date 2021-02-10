package factory

import (
	"cloudcadetest/framework/factory/platform"
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/module"
	"cloudcadetest/framework/rpc"
	"cloudcadetest/modconf"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"
)

type CServer struct {
	cfg     *modconf.ServerConf
	logger  *log.Logger
	stopped bool

	serverMod *module.ServerMod
}

func New(sc *modconf.ServerConf) *CServer {
	if sc == nil {
		panic("no conf provided")
	}

	return &CServer{
		cfg:     sc,
		logger:  nil,
		stopped: false,
		serverMod: &module.ServerMod{
			GoLen:               2048,
			TimerDispatcherLen:  2048,
			RPCServer:           rpc.NewServer(1000, 0),
			FuncGetExecTimeOut: func() int32 {
				return 10
			},
			FuncGetBlockTimeOut: func() int32 {
				return 10
			},
		},
	}
}

func (s *CServer) GetEntity() *module.ServerMod {
	return s.serverMod
}

func (s *CServer) Run(mods []module.IModule) {
	var err error

	fatalDir := s.cfg.LogPath + "/error"
	err = os.MkdirAll(fatalDir, 0777)
	if err != nil {
		panic(err.Error())
	}

	now := time.Now()
	fatalFileName := fmt.Sprintf(fatalDir+"/%d%02d%02d-%02d-%02d-%02d.log", now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())

	var logFile *os.File
	logFile, err = os.OpenFile(fatalFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
	if err == nil {
		// 重定向stderr
		platform.DupExt(int(logFile.Fd()), int(os.Stderr.Fd()))
	} else {
		panic(err.Error())
	}

	rand.Seed(time.Now().UnixNano())

	// logger
	if s.cfg.LogLevel != "" {
		var err error
		s.logger, err = log.New(s.cfg.LogLevel, s.cfg.LogPath, s.cfg.LogFileName, s.cfg.LogChanNum, s.cfg.RollSize)
		if err != nil {
			panic(err)
		}

		s.logger.EnableStdOut(s.cfg.EnableStdOut)

		log.Export(s.logger)
		defer log.Close()
	}

	for i := 0; i < len(mods); i++ {
		module.Run(mods[i])
	}

	log.Release("cc-server starting up")

	// 捕获信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, []os.Signal{os.Kill}...)

	for !s.stopped {
		select {
		case sig := <-c:
			switch sig {
			case os.Kill:
				log.Release("cc-server killed")
				goto END
			}
		}
	}

END:
	module.Destroy()
	log.Release("cc-server closing down")
}
