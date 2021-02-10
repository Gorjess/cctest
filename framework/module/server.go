package module

import (
	"cloudcadetest/conf"
	"cloudcadetest/framework/g"
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
	g                   *g.Go
	dispatcher          *timer.Dispatcher
	server              *rpc.Server
	FuncGetExecTimeOut  func() int32
	FuncGetBlockTimeOut func() int32
}

func (sm *ServerMod) Init(funcGetRpcExecTimeOut func() int32, funcGetBlockTimeOut func() int32) {
	if sm.GoLen <= 0 {
		sm.GoLen = 0
	}
	if sm.TimerDispatcherLen <= 0 {
		sm.TimerDispatcherLen = 0
	}

	sm.g = g.New(sm.GoLen)
	sm.dispatcher = timer.NewDispatcher(sm.TimerDispatcherLen)
	sm.server = sm.RPCServer
	sm.FuncGetExecTimeOut = funcGetRpcExecTimeOut
	sm.FuncGetBlockTimeOut = funcGetBlockTimeOut
	if sm.RPCServer != nil {
		sm.RPCServer.FuncGetExecTimeOut = funcGetRpcExecTimeOut
	}
	if sm.server == nil {
		sm.server = rpc.NewServer(0, 0)
	}
}

func (sm *ServerMod) runTimer(t *timer.Timer) {
	if t == nil {
		return
	}

	start := time.Now()
	defer func() {
		if sm.FuncGetExecTimeOut != nil {
			timeOut := sm.FuncGetExecTimeOut()
			if timeOut > 0 {
				dt := time.Since(start)
				if dt >= time.Duration(timeOut)*time.Millisecond {
					log.Warn("call info ddchess timeout f:%v t:%v", t.Name, dt)
				}
			}
		}

		var err error
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				err = fmt.Errorf("%v: %sm", r, buf[:l])
			} else {
				err = fmt.Errorf("%v", r)
			}

			log.Error("%v", err.Error())
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

func (sm *ServerMod) runChanCB(f func()) {
	start := time.Now()

	defer func() {
		sm.g.PendingGo--

		if sm.FuncGetExecTimeOut != nil {
			timeOut := sm.FuncGetExecTimeOut()
			if timeOut > 0 {
				dt := time.Since(start)
				if dt >= time.Duration(timeOut)*time.Millisecond {
					log.Warn("sm.runChanCB timeout, f:%v t:%sm", GetFunctionName(f), dt)
				}
			}
		}

		var err error
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				err = fmt.Errorf("%v: %sm", r, buf[:l])
			} else {
				err = fmt.Errorf("%v", r)
			}

			log.Error("sm.runChanCB:%sm", err.Error())
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

	timeout := 10
	if sm.FuncGetBlockTimeOut != nil {
		timeout = int(sm.FuncGetBlockTimeOut())
	}

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
			log.Release("ServerMod closing")
			sm.server.Close()
			log.Release("sm.server.Close()")
			sm.g.Close()
			log.Release("sm.g.Close()")
			return

		case ci := <-sm.server.ChanCall:
			sm.runFunc(ci.GetId(), func() {
				err := sm.server.Exec(ci)
				if err != nil {
					log.Error("%v", err)
				}
			})

		case cb := <-sm.g.ChanCb:
			sm.runFunc(nil, func() {
				sm.runChanCB(cb)
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

func (sm *ServerMod) Go(f func(), cb func()) {
	if sm.GoLen == 0 {
		panic("invalid GoLen")
	}

	sm.g.Go(f, cb)
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
