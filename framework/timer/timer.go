package timer

import (
	"cloudcadetest/conf"
	"cloudcadetest/framework/log"
	"runtime"
	"time"
)

// one dispatcher per goroutine (goroutine not safe)
type Dispatcher struct {
	ChanTimer chan *Timer
}

func NewDispatcher(l int) *Dispatcher {
	disp := new(Dispatcher)
	disp.ChanTimer = make(chan *Timer, l)
	return disp
}

// Timer
type Timer struct {
	Name string
	t    *time.Timer
	cb   func()
}

func (t *Timer) Stop() {
	t.t.Stop()
	t.cb = nil
}

func (t *Timer) CB() {
	defer func() {
		t.cb = nil
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("timer.CB:%v: %s", r, buf[:l])
			} else {
				log.Error("timer.CB:%v", r)
			}
		}
	}()

	if t.cb != nil {
		t.cb()
	}
}

func (disp *Dispatcher) AfterFunc(name string, d time.Duration, cb func()) *Timer {
	t := new(Timer)
	t.Name = name
	t.cb = cb
	t.t = time.AfterFunc(d, func() {
		disp.ChanTimer <- t
	})
	return t
}

// Cron
type Cron struct {
	t *Timer
}

func (c *Cron) Stop() {
	if c.t != nil {
		c.t.Stop()
	}
}

func (disp *Dispatcher) CronFunc(name string, cronExpr *CronExpr, _cb func()) *Cron {
	c := new(Cron)

	now := time.Now()
	nextTime := cronExpr.Next(now)
	if nextTime.IsZero() {
		return c
	}

	// callback
	var cb func()
	cb = func() {
		defer _cb()

		now := time.Now()
		nextTime := cronExpr.Next(now)
		if nextTime.IsZero() {
			return
		}
		c.t = disp.AfterFunc(name, nextTime.Sub(now), cb)
	}

	c.t = disp.AfterFunc(name, nextTime.Sub(now), cb)
	return c
}

// Ticker
type Ticker struct {
	stopped bool
	t       *Timer
}

func (t *Ticker) IsStopped() bool {
	return t.stopped
}

func (t *Ticker) Stop() {
	if t.t != nil {
		t.stopped = true
		t.t.Stop()
	}
}

func (disp *Dispatcher) NewTicker(name string, d time.Duration, _cb func()) *Ticker {
	if d == 0 {
		return nil
	}

	t := new(Ticker)
	t.stopped = false

	// callback
	var cb func()
	cb = func() {
		defer _cb()
		t.t = disp.AfterFunc(name, d, cb)
	}

	t.t = disp.AfterFunc(name, d, cb)
	return t
}
