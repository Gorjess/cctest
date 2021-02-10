package roommgr

import (
	"cloudcadetest/common/task"
	"cloudcadetest/common/uuid"
	"cloudcadetest/framework/log"
	"strconv"
)

type Manager struct {
	taskPool *task.Pool
	uuid *uuid.UUID
	rooms map[int64]*Room
}

func New() *Manager {
	return &Manager{
		taskPool: task.NewTaskPool(0, 0),
		uuid: &uuid.UUID{},
		rooms: map[int64]*Room{},
	}
}

func (m *Manager) AddRoom() int64 {
	id := m.uuid.Get()
	m.rooms[id] = NewRoom()
	return id
}

func (m *Manager) DeleteRoom(uuid int64) {
	delete(m.rooms, uuid)
}

func (m *Manager) AddRoomTask(roomID int64, f, cb func()) int {
	if m.taskPool == nil {
		log.Error("nil room task pool, room:%d", roomID)
		return -1
	}
	m.taskPool.AddTask(f, cb, strconv.FormatInt(roomID, 10))
	return 0
}
