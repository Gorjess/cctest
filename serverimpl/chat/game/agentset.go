package game

import (
	"sync"
)

type IPlayer interface {
	GetFD() int64
}

var (
	agentSet     = map[int64]IPlayer{}
	agentSetLock sync.Mutex
)

func AddAgentPlayer(p IPlayer) {
	if p == nil {
		return
	}

	agentSetLock.Lock()
	defer agentSetLock.Unlock()

	agentSet[p.GetFD()] = p
}

func GetAgentPlayer(fd int64) IPlayer {
	agentSetLock.Lock()
	defer agentSetLock.Unlock()

	return agentSet[fd]
}

func DelAgentPlayer(fd int64) {
	agentSetLock.Lock()
	defer agentSetLock.Unlock()

	delete(agentSet, fd)
}

func GetAgentPlayerCount() int {
	agentSetLock.Lock()
	defer agentSetLock.Unlock()

	return len(agentSet)
}
