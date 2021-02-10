package roommgr

import (
	"cloudcadetest/common/task"
	"cloudcadetest/common/uuid"
	"cloudcadetest/framework/log"
	"container/list"
	"errors"
	"fmt"
	"strconv"
)

type Manager struct {
	taskPool   *task.Pool
	uuid       *uuid.UUID
	rooms      map[int64]*Room
	validRooms *list.List
}

func New() *Manager {
	return &Manager{
		taskPool:   task.NewTaskPool(0, 0),
		uuid:       &uuid.UUID{},
		rooms:      map[int64]*Room{},
		validRooms: list.New(),
	}
}

type RoomState int

const (
	valid RoomState = iota
	invalid
	full
)

func (m *Manager) push(r *Room) {
	n := m.validRooms.PushBack(r.id)
	r.node = n
}

func (m *Manager) AddRoom() *Room {
	id := m.uuid.Get()
	r := NewRoom(id)
	m.push(r)
	m.rooms[id] = r
	return r
}

func (m *Manager) DeleteRoom(id int64) {
	r, ok := m.rooms[id]
	if !ok {
		return
	}
	m.validRooms.Remove(r.node)
	delete(m.rooms, id)
}

func (m *Manager) AddRoomTask(roomID int64, f, cb func()) int {
	if m.taskPool == nil {
		log.Error("nil room task pool, room:%d", roomID)
		return -1
	}
	m.taskPool.AddTask(f, cb, strconv.FormatInt(roomID, 10))
	return 0
}

func (m *Manager) Join(playerFD int64) (int64, error) {
	id := m.validRooms.Front()
	if id == nil {
		return -1, errors.New("no room valid")
	}

	r := m.rooms[id.Value.(int64)]
	state := r.Join(playerFD)
	if state == full {
		m.validRooms.Remove(r.node)
		r.node = nil
	}

	return r.id, nil
}

func (m *Manager) Leave(playerFD, roomID int64) error {
	r, ok := m.rooms[roomID]
	if !ok {
		return fmt.Errorf("room %d not found", roomID)
	}
	r.Leave(playerFD)
	if r.node == nil {
		m.push(r)
	}
	return nil
}
