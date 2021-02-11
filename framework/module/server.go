package module

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/rpc"
	"cloudcadetest/framework/timer"
	"context"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

type ServerMod struct {
	GoLen               int
	TimerDispatcherLen  int
	RPCServer           *rpc.Server
	dispatcher          *timer.Dispatcher
	server              *rpc.Server
}

func (sm *ServerMod) Init() {
	if sm.GoLen <= 0 {
		sm.GoLen = 0
	}
	if sm.TimerDispatcherLen <= 0 {
		sm.TimerDispatcherLen = 0
	}

	sm.dispatcher = timer.NewDispatcher(sm.TimerDispatcherLen)
	sm.server = sm.RPCServer
	if sm.server == nil {
		sm.server = rpc.NewServer(0)
	}
}

func (sm *ServerMod) runTimer(t *timer.Timer) {
	if t == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			dumpRecover(r)
		}
	}()

	t.CB()
}

// 获取函数名称
func GetFunctionName(i interface{}, seps ...rune) string {
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()

	fields := strings.FieldsFunc(fn, func(sep rune) bool {
		for _, s := range seps {
			if sep == s {
				return true
			}
		}
		return false
	})

	if size := len(fields); size > 0 {
		return fields[size-1]
	}
	return "invalid-func-name"
}

func dumpRecover(r interface{}) {
	var err error
	buf := make([]byte, 4096)
	l := runtime.Stack(buf, false)
	err = fmt.Errorf("%v: %sm", r, buf[:l])
	log.Error("sm.runChanCB:%sm", err.Error())
}

func (sm *ServerMod) runChanCB(f func()) {
	defer func() {
		if r := recover(); r != nil {
			dumpRecover(r)
		}
	}()

	if f != nil {
		f()
	}
}

func (sm *ServerMod) runFunc(info interface{}, f func()) {
	if f == nil {
		return
	}

	timeout := 1000
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	ch := make(chan int, 3)

	go func() {
		select {
		case <-ctx.Done():
			if info == nil {
				info = GetFunctionName(f)
			}

			errMsg := fmt.Sprintf("%v fatal error: blocked %d second\n", info, timeout)
			_, e := fmt.Fprintf(os.Stderr, errMsg)
			if e != nil {
				fmt.Printf("redirect err:%sm failed:%sm", errMsg, e.Error())
			}

			time.Sleep(3 * time.Second)
			os.Exit(-1)
		case <-ch:
		}
	}()

	f()

	ch <- 1
}

func (sm *ServerMod) Run(closeSig chan bool) {
	debug.SetPanicOnFault(true)

	for {
		select {
		case <-closeSig:
			log.Release("serverMod closing")
			sm.server.Close()
			log.Release("sm.server.Close()")
			return

		case ci := <-sm.server.ChanCall:
			sm.runFunc(ci.GetId(), func() {
				err := sm.server.Exec(ci)
				if err != nil {
					log.Error("%s", err.Error())
				}
			})

		case t := <-sm.dispatcher.ChanTimer:
			sm.runFunc(t.Name, func() {
				sm.runTimer(t)
			})
		}
	}
}

func (sm *ServerMod) GetRPCTaskNum() int {
	return len(sm.RPCServer.ChanCall)
}

func (sm *ServerMod) AfterFunc(name string, d time.Duration, cb func()) *timer.Timer {
	if sm.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return sm.dispatcher.AfterFunc(name, d, cb)
}

func (sm *ServerMod) NewTicker(name string, d time.Duration, cb func()) *timer.Ticker {
	if sm.TimerDispatcherLen == 0 {
		panic("invalid TimerDispatcherLen")
	}

	return sm.dispatcher.NewTicker(name, d, cb)
}

func (sm *ServerMod) RegisterChanRPC(id interface{}, f interface{}) {
	if sm.RPCServer == nil {
		panic("invalid RPCServer")
	}

	sm.server.Register(id, f)
}

func (sm *ServerMod) GoChanRPC(id interface{}, args ...interface{}) {
	if sm.RPCServer == nil {
		panic("invalid RPCServer")
	}

	sm.server.Go(id, args...)
}

func (sm *ServerMod) RunInSkeleton(id interface{}, f func()) {
	if sm.RPCServer == nil {
		panic("invalid RPCServer")
	}

	sm.server.GoFunc(id, f)
}

func (sm *ServerMod) IsRegister(id interface{}) bool {
	if sm.RPCServer == nil {
		panic("invalid RPCServer")
	}

	return sm.server.IsRegister(id)
}
