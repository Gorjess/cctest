package roommgr

import (
	"container/list"
)

type Room struct {
	id          int64
	node        *list.Element
	members     map[int64]struct{}
	historyMsgs *list.List
}

func NewRoom(id int64) *Room {
	return &Room{
		id:          id,
		historyMsgs: list.New(),
	}
}

func (r *Room) AddMsg(msg interface{}) int {
	msgCnt := r.historyMsgs.Len()
	// 超过上限，移除最早的一条消息
	if msgCnt >= 50 {
		head := r.historyMsgs.Front()
		r.historyMsgs.Remove(head)
		msgCnt -= 1
	}

	r.historyMsgs.PushBack(msg)
	return msgCnt + 1
}

func (r *Room) GetHistoryMsgs() *list.List {
	return r.historyMsgs
}

func (r *Room) Join(playerFD int64) RoomState {
	l := len(r.members)
	if l >= 100 {
		return invalid
	}

	r.members[playerFD] = struct{}{}
	if l+1 == 100 {
		return full
	}
	return valid
}

func (r *Room) Leave(playerFD int64) {
	delete(r.members, playerFD)
}
