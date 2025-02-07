package game

import (
	"cloudcadetest/common/task"
	"cloudcadetest/common/word/filter"
	"cloudcadetest/common/word/frequency"
	"cloudcadetest/common/word/frequency/wordmeta"
	"cloudcadetest/framework/log"
	"cloudcadetest/pb"
	"container/list"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Manager struct {
	*filterSkeleton
	taskPool      *task.Pool
	roomIDBase    int64
	rooms         map[int64]*Room
	validRooms    *list.List
	players       map[int64]*Agent
	playersByName map[string]*Agent
	filter        *filter.Filter
	names         map[string]struct{}
	wordFrequency *frequency.Frequency
}

func NewRoomMgr() *Manager {
	m := &Manager{
		taskPool:       task.NewTaskPool(SM, 0, 0),
		roomIDBase:     0,
		rooms:          map[int64]*Room{},
		names:          map[string]struct{}{},
		players:        map[int64]*Agent{},
		playersByName:  map[string]*Agent{},
		validRooms:     list.New(),
		filterSkeleton: NewFS(),
		wordFrequency:  frequency.New(),
	}
	m.filter = filter.New(m)
	return m
}

func (m *Manager) newTid() int64 {
	if m.roomIDBase == math.MaxInt64 {
		m.roomIDBase = 0
	}
	m.roomIDBase++
	return m.roomIDBase
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
	id := m.newTid()
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
		return -1, errors.New(fmt.Sprintf("duplicate name:%s, %v", username, m.names))
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

	r.broadcast(-1, pb.CSMsgID_NTF_ROOM_MEMBER_ONLINE, &pb.CSNtfBody{RoomMemberOnline: &pb.CSNtfRoomMemberOnline{
		RoomID:   r.id,
		Username: username,
	}})

	// history messages
	m.notifyHistoryMsgs(p.GetFD())

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

func (m *Manager) recordWordFrequency(word string) {
	m.wordFrequency.Add(word)
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
		m.execGM(content[1:], func(result string) {
			r.notifyRoomChat(-1, result)
		})
	} else {
		r.filter.Check(content, func(newStr string) {
			r.AddMsg(p.username, newStr)
			r.notifyRoomChat(playerFD, newStr)
		})
	}
}

func (m *Manager) execGM(cmd string, onFinish func(result string)) {
	ss := strings.Split(cmd, " ")
	if len(ss) != 2 {
		onFinish("invalid cmd")
	}

	cmd = ss[0]
	arg := ss[1]

	switch cmd {
	case "popular":
		secs, e := strconv.Atoi(arg)
		if e == nil {
			m.mostFrequentWord(secs, func(meta *wordmeta.Data, e error) {
				if e != nil {
					log.Error("get freq failed:%s", e.Error())
					return
				}
				if meta == nil {
					onFinish("")
				} else {
					onFinish(meta.Word)
				}
			})
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
			onFinish(formatDur(time.Now().Sub(p.LoginTime)))
		}
	default:
		onFinish("non-supported cmd")
	}
}

func (m *Manager) mostFrequentWord(lastNSeconds int, onFinish func(meta *wordmeta.Data, e error)) {
	m.wordFrequency.GetFrequencyByTime(lastNSeconds, func(meta *wordmeta.Data, e error) {
		if onFinish == nil {
			if e != nil {
				log.Error("get freq failed:%s", e.Error())
				return
			}
			if meta == nil {
				log.Error("nil meta")
				return
			}
		} else {
			onFinish(meta, e)
		}
	})
}

func (m *Manager) notifyHistoryMsgs(playerFD int64) {
	p := m.players[playerFD]
	if p == nil {
		return
	}
	r := m.rooms[p.GetRoomID()]
	if r == nil {
		return
	}

	var (
		ntfID    = pb.CSMsgID_NTF_HISTROY_MSG
		csNtf    = &pb.CSNtfBody{HistoryMsg: &pb.CSNtfHistoryMsg{}}
		msgCount = r.historyMsgs.Len()
	)

	// 整合消息
	csNtf.HistoryMsg.History = make([]*pb.HistoryChat, msgCount)
	n := r.historyMsgs.Front()
	if n == nil {
		return
	}
	i := 0
	for n := r.historyMsgs.Front(); n != nil; n = n.Next() {
		csNtf.HistoryMsg.History[i] = n.Value.(*pb.HistoryChat)
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

func (m *Manager) Chat(p *Agent, chat *pb.CSReqChat) error {
	otherp := m.playersByName[chat.Username]
	if otherp == nil {
		return errors.New("target player not found")
	}
	if p.roomID != otherp.roomID {
		return errors.New("not in one same room")
	}

	r := m.rooms[p.roomID]
	if r == nil {
		return errors.New("room entity not found")
	}

	m.AddRoomTask(p.roomID, func() {
		r.filter.Check(chat.Content, func(newStr string) {
			otherp.SendClient(pb.CSMsgID_NTF_CHAT, &pb.CSNtfBody{Chat: &pb.CSNtfChat{
				From:    p.username,
				Content: newStr,
			}}, nil)
		})
	}, nil)

	otherp.SendClient(pb.CSMsgID_NTF_CHAT, &pb.CSNtfBody{Chat: &pb.CSNtfChat{
		From:    p.username,
		Content: chat.Content,
	}}, nil)
	return nil
}
