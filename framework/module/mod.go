package module

import (
	"cloudcadetest/framework/log"
	"runtime"
	"sync"
)

type IModule interface {
	OnInit()
	OnDestroy()
	Run(closeSig chan bool)
}

type module struct {
	mi       IModule
	closeSig chan bool
	wg       sync.WaitGroup
}

var mods []*module

func register(mi IModule) *module {
	m := new(module)
	m.mi = mi
	m.closeSig = make(chan bool, 1)
	mods = append(mods, m)
	return m
}

func Destroy() {
	for i := len(mods) - 1; i >= 0; i-- {
		m := mods[i]
		m.closeSig <- true
		m.wg.Wait()
		destroy(m)
	}
}

func Run(mod IModule) {
	m := register(mod)
	mod.OnInit()

	go func() {
		m.wg.Add(1)
		m.mi.Run(m.closeSig)
		m.wg.Done()
	}()
}

func destroy(m *module) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			log.Error("%v: %s", r, buf[:l])
		}
	}()

	m.mi.OnDestroy()
}
