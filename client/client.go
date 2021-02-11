package main

import (
	"bufio"
	"bytes"
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
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

func NewTCPClient() {
	client := new(network.TCPClient)
	client.Addr = "127.0.0.1:3066"
	client.ConnectInterval = 3 * time.Second
	client.PendingWriteNum = 100
	client.NewAgent = NewPlayer
	client.DisconnectCB = func(s string) {
		log.Warn("disconnected:%s", s)
	}
	client.Start()
}

type Player struct {
	conn     network.IConn
	proc     *protobuf.Processor
	username string
}

func NewPlayer(conn *network.TCPConn) agent.Agent {
	p := &Player{
		conn: conn,
		proc: protobuf.NewProcessor(),
	}

	go p.conn.WriteTask()

	// login
	rand.Seed(time.Now().UnixNano())
	p.send(pb.CSMsgID_REQ_LOGIN, &pb.CSReqBody{Login: &pb.CSReqLogin{
		Username: "test_"+strconv.Itoa(rand.Intn(1000)),
	}})

	p.readTask()

	return p
}

func (p *Player) userInput(hint string, input chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(hint)
	scanner.Scan()
	input <- scanner.Text()
}

func (p *Player) OnClose(code uint) {
	log.Release("player destroyed:%d", code)
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
				log.Error("DealMsg %s Unmarshal fail[%s]", msgID, err.Error())
				return false
			}
			router(msgID, p, body)
			return true
		}

		if e := dealMsg(p.conn, recvBuffer, onceBuffer, msgHandler); e != nil {
			break
		}
	}

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
				log.Release("copy err:%v", h)
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

func register() {
	callbacks[pb.CSMsgID_RSP_LOGIN] = rspLogin
	callbacks[pb.CSMsgID_RSP_ROOM_LIST] = rspRoomList
	callbacks[pb.CSMsgID_RSP_JOIN_ROOM] = rspJoinRoom
	callbacks[pb.CSMsgID_NTF_ROOM_CHAT] = ntfRoomChat
	//callbacks[pb.CSMsgID_NTF_HISTROY_MSG] = ntfHistoryMsgs
}

func router(id pb.CSMsgID, args ...interface{}) {
	cb, ok := callbacks[id]
	if !ok {
		return
	}

	p := args[0].(*Player)
	body := args[1].(proto.Message)
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
		log.Warn("login failed:%s", rsp.ErrMsg)
		p.OnClose(1)
	} else {
		log.Release("finish logging in, room:%d", rsp.Login.RoomID)
		p.username = rsp.Login.Username

		time.Sleep(time.Second * 20)

		p.send(pb.CSMsgID_REQ_ROOM_CHAT, &pb.CSReqBody{
			RoomChat: &pb.CSReqRoomChat{Content: "hello"},
		})

		//go func() {
		//	sig := make(chan string, 1)
		//	p.userInput("say hello to everyone:", sig)
		//	content := <-sig
		//
		//	p.send(pb.CSMsgID_REQ_ROOM_CHAT, &pb.CSReqBody{
		//		RoomChat: &pb.CSReqRoomChat{Content: content},
		//	})
		//}()
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
	for _, room := range rsp.RoomList.Rooms {
		fmt.Printf("Room[%d]--(%d/%d)", room.RoomID, room.CurrentMemberNum, room.TotalMemberNum)
		rooms[room.RoomID] = struct{}{}
	}

	go func() {
		sig := make(chan string, 1)
		p.userInput("Input a room no:", sig)
		no, e := strconv.Atoi(<-sig)
		if e != nil {
			fmt.Println("invalid num")
			return
		}
		if _, ok := rooms[int64(no)]; !ok {
			fmt.Println("invalid num")
			return
		}

		p.send(pb.CSMsgID_REQ_JOIN_ROOM, &pb.CSReqBody{
			JoinRoom: &pb.CSReqJoinRoom{RoomID: int64(no)},
		})
	}()
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
		log.Warn("join room failed")
		return
	}

	log.Release("finish joining room")
}

func ntfRoomChat(p *Player, body interface{}) {
	ntf, ok := body.(*pb.CSNtfBody)
	if !ok {
		return
	}
	if ntf.RoomChat == nil {
		return
	}
	un := ntf.RoomChat.Username
	if un == p.username {
		un = "You"
	}
	log.Release("[%s] says: [%s]", un, ntf.RoomChat.Content)
}

func (p *Player) send(msgID pb.CSMsgID, body interface{}) {
	bodyData, e := p.proc.Marshal([]interface{}{int32(msgID), body})
	if e != nil {
		log.Error(e.Error())
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
		log.Error("Marshal head error:" + err.Error())
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
		log.Error("write %s failed:%s", msgID, e.Error())
	}

	log.Release("send msg:[%s][%v]", msgID, body)
}

func main() {
	register()

	NewTCPClient()
}
