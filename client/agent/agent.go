package agent

import (
	"bufio"
	"bytes"
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/network"
	"cloudcadetest/framework/network/protobuf"
	"cloudcadetest/pb"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func New() {
	registerHandlers()
	client := new(network.TCPClient)
	client.Addr = "127.0.0.1:3066"
	client.ConnectInterval = 3 * time.Second
	client.PendingWriteNum = 100
	client.NewAgent = NewPlayer
	client.DisconnectCB = func(s string) {
		pureLog("disconnected:%s", s)
	}
	client.Start()
}

type chat struct {
	content string
	from    string
	ts      time.Time
}

type Player struct {
	conn     network.IConn
	proc     *protobuf.Processor
	username string
	chats    chan *chat
}

func NewPlayer(conn *network.TCPConn) agent.Agent {
	printTitle()

	p := &Player{
		conn:  conn,
		proc:  protobuf.NewProcessor(),
		chats: make(chan *chat, 100),
	}

	go p.conn.WriteTask()
	go p.handleChats()

	p.login()

	p.readTask()

	return p
}

func (p *Player) userInput(hint string, cb func(string)) {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println(hint)
		scanner.Scan()
		cb(scanner.Text())
	}()
}

func (p *Player) handleChats() {
	for {
		select {
		case c := <-p.chats:
			printChat(p.username, c.from, c.content, c.ts)
		default:
		}
	}
}

func (p *Player) login() {
	rand.Seed(time.Now().UnixNano())
	p.send(pb.CSMsgID_REQ_LOGIN, &pb.CSReqBody{Login: &pb.CSReqLogin{
		Username: "test_" + strconv.Itoa(rand.Intn(1000)),
	}})
}

func (p *Player) OnClose(code uint) {
	pureLog("player destroyed:%d", code)
	p.conn.Close()
	os.Exit(-1)
}

func (p *Player) Addr() string {
	return p.conn.LocalAddr().String()
}

func (p *Player) readTask() {
	recvBuffer := new(bytes.Buffer)
	onceBuffer := make([]byte, 4096)
	for {
		msgHandler := func(msgID pb.CSMsgID, bodyBuf []byte) bool {
			var body proto.Message
			if msgID < pb.CSMsgID_NTF_BEGIN {
				body = &pb.CSRspBody{}
			} else {
				body = &pb.CSNtfBody{}
			}
			err := proto.Unmarshal(bodyBuf, body)
			if err != nil {
				pureLog("DealMsg %s Unmarshal fail[%s]", msgID, err.Error())
				return false
			}
			router(msgID, p, body)
			return true
		}

		if e := dealMsg(p.conn, recvBuffer, onceBuffer, msgHandler); e != nil {
			break
		}
	}

	//close(taskReady)

	p.OnClose(uint(999))
}

func dealMsg(conn network.IConn, recvBuffer *bytes.Buffer, onceBuffer []byte, msgHandler func(pb.CSMsgID, []byte) bool) error {
	// 从网络层读取数据
	n, err := conn.Read(onceBuffer)
	if err != nil {
		return err
	}

	// 将数据串起来,方便处理粘包
	_, err = recvBuffer.Write(onceBuffer[:n])
	if err != nil {
		return err
	}

	// 处理消息包(处理粘包 一次最大处理16个包)
	for i := 0; i < 16; i++ {
		rlen := recvBuffer.Len()
		if rlen < 1 { // 包不够长度
			break
		}

		buf := recvBuffer.Bytes()
		hlen := int32(buf[0])
		if hlen > 1000 {
			return fmt.Errorf("message too long %d", hlen)
		} else if hlen <= 1 {
			return fmt.Errorf("message too short %d", hlen)
		}

		if rlen < int(hlen)+1 {
			break
		}

		//parse head
		h := &pb.CSHead{}
		if err = proto.Unmarshal(buf[1:hlen+1], h); err != nil {
			return errors.New("Unmarshal head error:" + err.Error() + fmt.Sprintf(" headLen:%v", hlen))
		}

		if h.BodyLen > 1000 {
			return fmt.Errorf("message too long %v", h.BodyLen)
		} else if h.BodyLen < 1 {
			return fmt.Errorf("message too short %v", h.BodyLen)
		}

		pktLen := int(hlen) + 1 + int(h.BodyLen)
		if rlen < pktLen {
			break
		}

		data := recvBuffer.Next(pktLen)
		var bodyBuf []byte
		if h.BodyLen > 0 {
			bodyBuf = make([]byte, h.BodyLen, h.BodyLen)
			if copy(bodyBuf, data[int(hlen)+1:pktLen]) != int(h.BodyLen) { // 拷贝出错了
				pureLog("copy err:%v", h)
				return errors.New("copy err")
			}

			if h.IsCompressed {
				//todo 需要解压缩
			}
		}

		if msgHandler != nil {
			if !msgHandler(h.MsgID, bodyBuf) {
				return errors.New("msg handler err")
			}
		}
	}
	return nil
}

var callbacks = map[pb.CSMsgID]func(*Player, interface{}){}

func registerHandlers() {
	callbacks[pb.CSMsgID_RSP_LOGIN] = rspLogin
	callbacks[pb.CSMsgID_RSP_ROOM_LIST] = rspRoomList
	callbacks[pb.CSMsgID_RSP_JOIN_ROOM] = rspJoinRoom
	callbacks[pb.CSMsgID_RSP_ROOM_CHAT] = rspRoomChat

	callbacks[pb.CSMsgID_NTF_ROOM_CHAT] = ntfRoomChat
	callbacks[pb.CSMsgID_NTF_HISTROY_MSG] = ntfHistoryMsgs
}

func router(id pb.CSMsgID, args ...interface{}) {
	cb, ok := callbacks[id]
	if !ok {
		return
	}

	p := args[0].(*Player)
	body := args[1]
	cb(p, body)
}

func rspLogin(p *Player, body interface{}) {
	rsp, ok := body.(*pb.CSRspBody)
	if !ok {
		return
	}
	if rsp.Login == nil {
		return
	}

	if rsp.ErrCode != pb.ERROR_CODE_SUCCESS {
		pureLog("login failed:%s", rsp.ErrMsg)
		p.OnClose(1)
	} else {
		p.username = rsp.Login.Username

		p.userInput("say something:", func(input string) {
			p.send(pb.CSMsgID_REQ_ROOM_CHAT, &pb.CSReqBody{
				RoomChat: &pb.CSReqRoomChat{Content: input},
			})
		})
	}
}

func rspRoomList(p *Player, body interface{}) {
	rsp, ok := body.(*pb.CSRspBody)
	if !ok {
		return
	}
	if rsp.RoomList == nil {
		return
	}

	rooms := map[int64]struct{}{}
	pureLog(`
                 ROOMS(%d):
--------------------------------------------`, len(rsp.RoomList.Rooms))

	for _, room := range rsp.RoomList.Rooms {
		pureLog("|%d| -- ( %d/%d )", room.RoomID, room.CurrentMemberNum, room.TotalMemberNum)
		rooms[room.RoomID] = struct{}{}
	}

	time.Sleep(time.Second * 2)
	p.send(pb.CSMsgID_REQ_JOIN_ROOM, &pb.CSReqBody{
		JoinRoom: &pb.CSReqJoinRoom{RoomID: int64(1)},
	})

	return
}

func rspJoinRoom(p *Player, body interface{}) {
	rsp, ok := body.(*pb.CSRspBody)
	if !ok {
		return
	}
	if rsp.JoinRoom == nil {
		return
	}

	if rsp.ErrCode != pb.ERROR_CODE_SUCCESS {
		pureLog("join room failed:%s", rsp.ErrMsg)
		return
	}

	pureLog("finish joining room")
}

func rspRoomChat(p *Player, body interface{}) {
	rsp, ok := body.(*pb.CSRspBody)
	if !ok {
		return
	}

	if rsp.RoomChat == nil {
		return
	}
}

func ntfRoomChat(p *Player, body interface{}) {
	ntf, ok := body.(*pb.CSNtfBody)
	if !ok {
		return
	}
	if ntf.RoomChat == nil {
		return
	}
	p.chats <- &chat{
		content: ntf.RoomChat.Content,
		from:    ntf.RoomChat.Username,
		ts:      time.Now(),
	}
}

func ntfHistoryMsgs(p *Player, body interface{}) {
	ntf, ok := body.(*pb.CSNtfBody)
	if !ok {
		return
	}

	if ntf.HistoryMsg == nil {
		return
	}

}

func (p *Player) send(msgID pb.CSMsgID, body interface{}) {
	bodyData, e := p.proc.Marshal([]interface{}{int32(msgID), body})
	if e != nil {
		pureLog(e.Error())
		return
	}

	bodyLen := int32(len(bodyData))

	h := &pb.CSHead{
		MsgID:        msgID,
		BodyLen:      bodyLen,
		IsCompressed: false,
	}

	headData, err := p.proc.Marshal([]interface{}{int32(msgID), h})
	if err != nil {
		pureLog("Marshal head error:" + err.Error())
		return
	}
	headLen := int32(len(headData))

	data := make([]byte, 1+headLen+bodyLen)
	data[0] = byte(headLen)

	copy(data[1:], headData)
	if bodyLen != 0 {
		copy(data[1+headLen:], bodyData)
	}

	e = p.conn.Write(data)
	if e != nil {
		pureLog("write %s failed:%s", msgID, e.Error())
	}
}
