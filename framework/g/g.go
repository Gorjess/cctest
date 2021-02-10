package g

import (
	"cloudcadetest/conf"
	"cloudcadetest/framework/log"
	"container/list"
	"runtime"
	"sync"
)

// one Go per goroutine (goroutine not safe)
type Go struct {
	ChanCb    chan func()
	PendingGo int
}

type LinearGo struct {
	f  func()
	cb func()
}

type LinearContext struct {
	g              *Go
	linearGo       *list.List
	mutexLinearGo  sync.Mutex
	mutexExecution sync.Mutex
}

func New(l int) *Go {
	g := new(Go)
	g.ChanCb = make(chan func(), l)
	return g
}

func (g *Go) Go(f func(), cb func()) {
	g.PendingGo++

	go func() {
		defer func() {
			g.ChanCb <- cb
			if r := recover(); r != nil {
				if conf.LenStackBuf > 0 {
					buf := make([]byte, conf.LenStackBuf)
					l := runtime.Stack(buf, false)
					log.Error("%v: %s", r, buf[:l])
				} else {
					log.Error("%v", r)
				}
			}
		}()

		f()
	}()
}

func (g *Go) Cb(cb func()) {
	defer func() {
		g.PendingGo--
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("%v: %s", r, buf[:l])
			} else {
				log.Error("%v", r)
			}
		}
	}()

	if cb != nil {
		cb()
	}
}

func (g *Go) Close() {
	for g.PendingGo > 0 {
		g.Cb(<-g.ChanCb)
	}
}

func (g *Go) NewLinearContext() *LinearContext {
	c := new(LinearContext)
	c.g = g
	c.linearGo = list.New()
	return c
}

func (c *LinearContext) Go(f func(), cb func()) {
	c.g.PendingGo++

	c.mutexLinearGo.Lock()
	c.linearGo.PushBack(&LinearGo{f: f, cb: cb})
	c.mutexLinearGo.Unlock()

	go func() {
		c.mutexExecution.Lock()
		defer c.mutexExecution.Unlock()

		c.mutexLinearGo.Lock()
		e := c.linearGo.Remove(c.linearGo.Front()).(*LinearGo)
		c.mutexLinearGo.Unlock()

		defer func() {
			c.g.ChanCb <- e.cb
			if r := recover(); r != nil {
				if conf.LenStackBuf > 0 {
					buf := make([]byte, conf.LenStackBuf)
					l := runtime.Stack(buf, false)
					log.Error("%v: %s", r, buf[:l])
				} else {
					log.Error("%v", r)
				}
			}
		}()

		e.f()
	}()
}
