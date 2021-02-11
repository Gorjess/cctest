package game

import (
	"cloudcadetest/common/uuid"
	"cloudcadetest/framework/module"
	"cloudcadetest/framework/msg/cs"
)

var (
	SM          *module.ServerMod
	CSProcessor *cs.Processor
	UUID        *uuid.UUID
	RoomMgr     *Manager
)

func Init(sm *module.ServerMod) {
	SM = sm
	CSProcessor = cs.New(sm, true, 10000, 1024, false)
	UUID = &uuid.UUID{}
	RoomMgr = NewRoomMgr()

	registerHandler()
}
