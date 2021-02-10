package roommgr

import "container/list"

type Room struct {
	historyMsgs *list.List
}

func NewRoom() *Room {
	return &Room{
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
	return msgCnt+1
}

func (r *Room) GetHistoryMsgs() *list.List {
	return r.historyMsgs
}




