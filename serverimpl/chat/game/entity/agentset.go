package entity

import (
	"cloudcadetest/serverimpl/chat/game/entity/player"
	"sync"
)

var (
	agentSet     = map[int64]*player.Player{}
	agentSetLock sync.Mutex
)

func AddAgentPlayer(p *player.Player) {
	if p == nil {
		return
	}

	agentSetLock.Lock()
	defer agentSetLock.Unlock()

	agentSet[p.FD] = p
}

func GetAgentPlayer(fd int64) *player.Player {
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
