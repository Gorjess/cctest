package game

import (
	"cloudcadetest/common/task"
	"cloudcadetest/common/uuid"
	"cloudcadetest/common/wordfilter"
	"cloudcadetest/framework/log"
	"cloudcadetest/pb"
	"container/list"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	taskPool      *task.Pool
	uuid          *uuid.UUID
	rooms         map[int64]*Room
	validRooms    *list.List
	players       map[int64]*Agent
	playersByName map[string]*Agent
	filter        *wordfilter.Filter
	names         map[string]struct{}
}

func NewRoomMgr() *Manager {
	return &Manager{
		taskPool:      task.NewTaskPool(SM, 0, 0),
		uuid:          &uuid.UUID{},
		rooms:         map[int64]*Room{},
		names:         map[string]struct{}{},
		players:       map[int64]*Agent{},
		playersByName: map[string]*Agent{},
		validRooms:    list.New(),
		filter: wordfilter.New(SM, 1),
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
		log.Error("nil task pool, room:%d", roomID)
		return -1
	}
	m.taskPool.AddTask(f, cb, strconv.FormatInt(roomID, 10))
	return 0
}

func (m *Manager) Join(p *Agent, username string) (int64, error) {
	if _, ok := m.names[username]; ok {
		return -1, errors.New(fmt.Sprintf("duplicated name:%s, %v", username, m.names))
	}

	m.names[username] = struct{}{}

	var r *Room
	id := m.validRooms.Front()
	if id == nil {
		if len(m.rooms) < 100 {
			r = m.AddRoom()
		} else {
			return -1, errors.New("no room valid")
		}
	} else {
		r = m.rooms[id.Value.(int64)]
	}

	state := r.Join(p.GetFD())
	if state == full {
		m.validRooms.Remove(r.node)
		r.node = nil
	}
	p.SetRoomID(r.id)
	m.players[p.GetFD()] = p
	m.playersByName[username] = p

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
	p := m.players[playerFD]
	if p == nil {
		return errors.New("player not found")
	}
	delete(m.players, playerFD)
	delete(m.playersByName, p.GetUsername())

	return nil
}

func (m *Manager) RoomChat(playerFD, roomID int64, content string) {
	p := m.players[playerFD]
	if p == nil {
		return
	}
	r := m.rooms[roomID]
	if r == nil {
		return
	}

	// GM
	if strings.Index(content, "/") == 0 {
		content = m.execGM(r, content[1:])
		r.broadcast(-1, content)
	} else {
		r.filter.Check(content, func(newStr string) {
			r.AddMsg(newStr)
			r.broadcast(playerFD, newStr)
		})
	}
}

func (m *Manager) execGM(r *Room, cmd string) string {
	ss := strings.Split(cmd, " ")
	if len(ss) != 2 {
		return ""
	}

	cmd = ss[0]
	arg := ss[1]

	switch cmd {
	case "popular":
		secs, e := strconv.Atoi(arg)
		if e == nil {
			return r.mostFrequentWord(secs)
		}
	case "stats":
		p := m.playersByName[arg]
		if p != nil {
			formatDur := func(d time.Duration) string {
				dur := d.Round(time.Hour)
				day := d / 24
				h := day - day*24
				dur -= dur * time.Hour
				min := d / time.Minute
				d -= min * time.Minute
				return fmt.Sprintf("%02d %02d %02d %02d", day, h, min, d)
			}
			return formatDur(time.Now().Sub(p.LoginTime))
		}
	}

	return ""
}

func (m *Manager) NotifyHistoryMsgs(playerFD int64) {
	p := m.players[playerFD]
	if p == nil {
		return
	}
	r := m.rooms[p.GetRoomID()]
	if r == nil {
		return
	}

	var (
		ntfID = pb.CSMsgID_NTF_HISTROY_MSG
		csNtf = &pb.CSNtfBody{HistoryMsg: &pb.CSNtfHistoryMsg{
		}}
		msgCount = r.historyMsgs.Len()
	)

	// 整合消息
	csNtf.HistoryMsg.History = make([]string, msgCount)
	n := r.historyMsgs.Front()
	if n == nil {
		return
	}
	i := 0
	for n := r.historyMsgs.Front(); n != nil; n = n.Next() {
		csNtf.HistoryMsg.History[i] = n.Value.(string)
		i++
	}

	p.SendClient(ntfID, csNtf, nil)
}

func (m *Manager) SetName(p *Agent, name string, onFinish func(passed string)) {
	if p == nil {
		return
	}
	if _, ok := m.names[name]; ok {
		return
	}

	m.filter.Check(name, func(newStr string) {
		if onFinish != nil {
			onFinish(newStr)
		}
	})
}

func (m *Manager) GetRoomList(maxCount int32) []*pb.RoomInfo {
	ret := make([]*pb.RoomInfo, 0, maxCount)
	for id, r := range m.rooms {
		ri := &pb.RoomInfo{
			CurrentMemberNum: int32(len(r.members)),
			TotalMemberNum:   100,
			RoomID:           id,
		}
		ret = append(ret, ri)
	}
	return ret
}
