package game

import (
	"cloudcadetest/common/uuid"
	"cloudcadetest/framework/factory"
	"cloudcadetest/framework/msg/cs"
	"cloudcadetest/serverimpl/chat/game/roommgr"
)

var (
	Server      *factory.CServer
	CSProcessor *cs.Processor
	UUID        *uuid.UUID
	RoomMgr     *roommgr.Manager
)

func Init() {
	CSProcessor = cs.New(Server.GetEntity(), true, 10000, 1024, false)
	UUID = &uuid.UUID{}
	RoomMgr = roommgr.New()
}
