package rpc

import (
	"cloudcadetest/framework/log"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"time"
)

// one server per goroutine (goroutine not safe)
// one client per goroutine (goroutine not safe)
type Server struct {
	// id -> function
	//
	// function:
	// func(args []interface{})
	// func(args []interface{}) interface{}
	// func(args []interface{}) []interface{}
	functions          map[interface{}]interface{}
	ChanCall           chan *CallInfo
	FuncGetExecTimeOut func() int32
	ClearChanCallFlag  uint32
	LastFullChanTime   int64
	enableFailFast     uint32
	warnFull           uint32
}

type CallInfo struct {
	id      interface{}
	f       interface{}
	args    []interface{}
	chanRet chan *RetInfo
	cb      interface{}
}

func (ci *CallInfo) GetId() interface{} {
	return ci.id
}

type RetInfo struct {
	// nil
	// interface{}
	// []interface{}
	ret interface{}
	err error
	// callback:
	// func(err error)
	// func(ret interface{}, err error)
	// func(ret []interface{}, err error)
	cb interface{}
}

type Client struct {
	s               *Server
	chanSyncRet     chan *RetInfo
	ChanAsynRet     chan *RetInfo
	pendingAsynCall int
}

func NewServer(l int, enableFailFast uint32) *Server {
	s := new(Server)
	s.functions = make(map[interface{}]interface{})
	s.ChanCall = make(chan *CallInfo, l)
	atomic.StoreUint32(&s.enableFailFast, enableFailFast)
	nowUnix := time.Now().Unix()
	atomic.StoreInt64(&s.LastFullChanTime, nowUnix)
	return s
}

// call Register before calling Open and Go
func (s *Server) Register(id interface{}, f interface{}) {
	switch f.(type) {
	case func([]interface{}):
	case func([]interface{}) interface{}:
	case func([]interface{}) []interface{}:
	default:
		panic(fmt.Sprintf("function id %v: definition of function is invalid", id))
	}

	if _, ok := s.functions[id]; ok {
		panic(fmt.Sprintf("function id %v: already registered", id))
	}

	s.functions[id] = f
}

func (s *Server) IsRegister(id interface{}) bool {
	_, ok := s.functions[id]
	return ok
}

func (s *Server) ret(ci *CallInfo, ri *RetInfo) (err error) {
	if ci.chanRet == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	ri.cb = ci.cb
	ci.chanRet <- ri
	return
}

func doRecover(r interface{}) {
	buf := make([]byte, 4096)
	l := runtime.Stack(buf, false)
	log.Error("%v: %s", r, buf[:l])
}

func (s *Server) Exec(ci *CallInfo) (err error) {
	start := time.Now()
	defer func() {
		if s.FuncGetExecTimeOut != nil {
			timeOut := s.FuncGetExecTimeOut()
			if timeOut > 0 {
				dt := time.Since(start)
				if dt >= time.Duration(timeOut)*time.Millisecond {
					log.Warn("call info ddchess timeout f:%v t:%v", ci.id, dt)
				}
			}
		}

		if r := recover(); r != nil {
			doRecover(r)
			err = s.ret(ci, &RetInfo{err: fmt.Errorf("%v", r)})
			if err != nil {
				log.Error("server.ret:%s", err.Error())
			}
		}
	}()

	// execute
	switch ci.f.(type) {
	case func([]interface{}):
		ci.f.(func([]interface{}))(ci.args)
		return s.ret(ci, &RetInfo{})
	case func([]interface{}) interface{}:
		ret := ci.f.(func([]interface{}) interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})
	case func([]interface{}) []interface{}:
		ret := ci.f.(func([]interface{}) []interface{})(ci.args)
		return s.ret(ci, &RetInfo{ret: ret})

	case func():
		ci.f.(func())()
		return s.ret(ci, &RetInfo{})

	default:
		return fmt.Errorf("unknown func %v %v", ci.f, ci)
	}
}

func (s *Server) CheckFailFast() {
	if atomic.LoadUint32(&s.enableFailFast) == 0 {
		return
	}

	nowUnix := time.Now().Unix()
	chanCallCap := len(s.ChanCall)

	if nowUnix-atomic.LoadInt64(&s.LastFullChanTime) >= 30 {
		log.Error("FailFast")
		atomic.StoreUint32(&s.ClearChanCallFlag, 1)
		for i := 0; i < chanCallCap; i++ {
			select {
			case <-s.ChanCall:
			default:
				break
			}
		}

		log.Error("FailFast Over")

		atomic.StoreInt64(&s.LastFullChanTime, nowUnix)
		atomic.StoreUint32(&s.ClearChanCallFlag, 0)
		atomic.StoreUint32(&s.warnFull, 0)
	}
}

func (s *Server) AddChanCall(callInfo *CallInfo) {
	if callInfo == nil {
		return
	}

	select {
	case s.ChanCall <- callInfo:
		log.Debug("callinfo:%v", callInfo.id)
	default:
		log.Error("RPC ChanCall is full")
	}
}

// goroutine safe
func (s *Server) Go(id interface{}, args ...interface{}) {
	f := s.functions[id]
	if f == nil {
		log.Warn("id[%v] is not register", id)
		return
	}

	if atomic.LoadUint32(&s.enableFailFast) != 0 && atomic.LoadUint32(&s.ClearChanCallFlag) != 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 1<<20)
			l := runtime.Stack(buf, false)
			log.Error("%v: %s", r, buf[:l])
		}
	}()

	nowUnix := time.Now().Unix()

	if atomic.LoadUint32(&s.enableFailFast) != 0 && atomic.LoadUint32(&s.warnFull) != 0 {
		s.CheckFailFast()
	}

	ChanCallCap := cap(s.ChanCall)
	ChanCallLen := len(s.ChanCall)
	if ChanCallCap == ChanCallLen {
		log.Error("RPC ChanCall is full!!!!")
		if atomic.LoadUint32(&s.enableFailFast) != 0 && atomic.LoadUint32(&s.warnFull) == 0 {
			atomic.StoreInt64(&s.LastFullChanTime, nowUnix)
			atomic.StoreUint32(&s.warnFull, 1)
		}
		return
	}

	if atomic.LoadUint32(&s.enableFailFast) != 0 {
		if ChanCallLen < int(float64(ChanCallCap)*0.9) {
			atomic.StoreUint32(&s.warnFull, 0)
		}
	}

	s.AddChanCall(&CallInfo{
		id:   id,
		f:    f,
		args: args,
	})
}

func (s *Server) GoFunc(id interface{}, f func()) {
	defer func() {
		recover()
	}()

	s.AddChanCall(&CallInfo{
		id: id,
		f:  f,
	})
}

func (s *Server) Close() {
	close(s.ChanCall)

	var e error
	for ci := range s.ChanCall {
		e = s.ret(ci, &RetInfo{
			err: errors.New("chanrpc server closed"),
		})
		if e != nil {
			log.Error("server.ret:%s", e.Error())
		}
	}
}

// goroutine safe
func (s *Server) Open(l int) *Client {
	c := new(Client)
	c.s = s
	c.chanSyncRet = make(chan *RetInfo, 1)
	c.ChanAsynRet = make(chan *RetInfo, l)
	return c
}

func (c *Client) call(ci *CallInfo, block bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	if block {
		c.s.ChanCall <- ci
	} else {
		c.s.AddChanCall(ci)
	}
	return
}

func (c *Client) f(id interface{}, n int) (f interface{}, err error) {
	f = c.s.functions[id]
	if f == nil {
		err = fmt.Errorf("function id %v: function not registered", id)
		return
	}

	var ok bool
	switch n {
	case 0:
		_, ok = f.(func([]interface{}))
	case 1:
		_, ok = f.(func([]interface{}) interface{})
	case 2:
		_, ok = f.(func([]interface{}) []interface{})
	default:
		panic("bug")
	}

	if !ok {
		err = fmt.Errorf("function id %v: mismatched return type", id)
	}
	return
}

func (c *Client) Call0(id interface{}, args ...interface{}) error {
	f, err := c.f(id, 0)
	if err != nil {
		return err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return err
	}

	ri := <-c.chanSyncRet
	return ri.err
}

func (c *Client) Call1(id interface{}, args ...interface{}) (interface{}, error) {
	f, err := c.f(id, 1)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet
	return ri.ret, ri.err
}

func (c *Client) CallN(id interface{}, args ...interface{}) ([]interface{}, error) {
	f, err := c.f(id, 2)
	if err != nil {
		return nil, err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.chanSyncRet,
	}, true)
	if err != nil {
		return nil, err
	}

	ri := <-c.chanSyncRet
	return ri.ret.([]interface{}), ri.err
}

func (c *Client) asynCall(id interface{}, args []interface{}, cb interface{}, n int) error {
	f, err := c.f(id, n)
	if err != nil {
		return err
	}

	err = c.call(&CallInfo{
		f:       f,
		args:    args,
		chanRet: c.ChanAsynRet,
		cb:      cb,
	}, false)
	if err != nil {
		return err
	}

	c.pendingAsynCall++
	return nil
}

func (c *Client) AsynCall(id interface{}, _args ...interface{}) {
	if len(_args) < 1 {
		panic("callback function not found")
	}

	// args
	var args []interface{}
	if len(_args) > 1 {
		args = _args[:len(_args)-1]
	}

	// cb
	cb := _args[len(_args)-1]
	switch cb.(type) {
	case func(error):
		err := c.asynCall(id, args, cb, 0)
		if err != nil {
			cb.(func(error))(err)
		}
	case func(interface{}, error):
		err := c.asynCall(id, args, cb, 1)
		if err != nil {
			cb.(func(interface{}, error))(nil, err)
		}
	case func([]interface{}, error):
		err := c.asynCall(id, args, cb, 2)
		if err != nil {
			cb.(func([]interface{}, error))(nil, err)
		}
	default:
		panic("definition of callback function is invalid")
	}
}

func (c *Client) Cb(ri *RetInfo) {
	switch ri.cb.(type) {
	case func(error):
		ri.cb.(func(error))(ri.err)
	case func(interface{}, error):
		ri.cb.(func(interface{}, error))(ri.ret, ri.err)
	case func([]interface{}, error):
		ri.cb.(func([]interface{}, error))(ri.ret.([]interface{}), ri.err)
	default:
		panic("bug")
	}

	c.pendingAsynCall--
}

func (c *Client) Close() {
	for c.pendingAsynCall > 0 {
		c.Cb(<-c.ChanAsynRet)
	}
}
