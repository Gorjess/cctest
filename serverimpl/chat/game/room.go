package game

import (
	"cloudcadetest/common/wordfilter"
	"cloudcadetest/framework/log"
	"cloudcadetest/pb"
	"container/list"
)

type Room struct {
	id          int64
	node        *list.Element
	members     map[int64]struct{}
	historyMsgs *list.List
	filter      *wordfilter.Filter
}

func NewRoom(id int64) *Room {
	return &Room{
		id:          id,
		historyMsgs: list.New(),
		members:     map[int64]struct{}{},
		filter:      wordfilter.New(SM, id),
	}
}

func (r *Room) AddMsg(msg string) int {
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

func (r *Room) broadcast(playerFD int64, content string) {
	username := "N/A"
	p := RoomMgr.players[playerFD]
	if p != nil {
		username = p.GetUsername()
	} else {
		playerFD = -1
	}

	var (
		msgID = pb.CSMsgID_NTF_ROOM_CHAT
		csNtf = &pb.CSNtfBody{RoomChat: &pb.CSNtfRoomChat{
			Username: username,
			Content:  content,
		}}
	)
	RoomMgr.AddRoomTask(
		r.id,
		func() {
			compressedData, isCompressed, er := CSProcessor.CompressMsg(msgID, csNtf)
			if er != nil {
				log.Error("compress msg failed, id:%s, er:%s", msgID, er.Error())
				return
			}

			for fd := range r.members {
				//if fd == playerFD {
				//	continue
				//}
				mem := RoomMgr.players[fd]
				if mem == nil {
					continue
				}
				er = CSProcessor.Write2Socket(mem.GetConn(), msgID, compressedData, isCompressed, p.GetEncKey())
				if er != nil {
					log.Error("broadcast failed, room:%d, msg:%s, p:%d", r.id, msgID, fd)
				}
			}
		}, nil,
	)
}

func (r *Room) mostFrequentWord(bySeconds int) string {
	return ""
}
