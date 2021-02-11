package game

import (
	"bytes"
	"cloudcadetest/common/encrypt/aes"
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/network"
	"cloudcadetest/pb"
	"cloudcadetest/serverimpl/chat/conf"
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"time"
)

type Agent struct {
	conn   network.IConn //玩家连接
	fd     int64
	roomID int64

	username   string
	LoginTime  time.Time
	activeTime time.Time //活跃时间
	destroyed  bool      //已销毁标志
	encryptKey aes.Key
	working    bool //标识连接状态(false 等待客户端发送第一个包 true 收到客户端第一个包后进入工作模式)
}

func NewPlayer(conn *network.TCPConn) agent.Agent {
	if conn == nil {
		log.Error("conn is nil")
		return nil
	}

	now := time.Now()
	p := &Agent{
		conn:       conn,
		fd:         UUID.Get(),
		activeTime: now,
		LoginTime:  now,
	}

	SM.RunInSkeleton("gate.new.agent", func() {
		AddAgentPlayer(p)
	})

	log.Release("player[%s][%d] connected", p.conn.RemoteAddr(), p.fd)

	go p.conn.WriteTask()

	//处理读任务
	p.readTask()
	return p
}

func (p *Agent) GetFD() int64 {
	return p.fd
}

func (p *Agent) SetRoomID(id int64) {
	p.roomID = id
}

func (p *Agent) GetRoomID() int64 {
	return p.roomID
}

func (p *Agent) GetConn() network.IConn {
	return p.conn
}

func (p *Agent) IsDestroyed() bool {
	return p.destroyed
}

func (p *Agent) UpdateActiveTime(t time.Time) {
	p.activeTime = t
}

func (p *Agent) GetEncKey() *aes.Key {
	return &p.encryptKey
}

func (p *Agent) GetUsername() string {
	return p.username
}

func (p *Agent) SetUsername(name string) {
	p.username = name
}

func (p *Agent) who() string {
	return fmt.Sprintf("[player:%d,%s|room:%d|]", p.fd, p.username, p.roomID)
}

func (p *Agent) LogRelease(format string, args ...interface{}) {
	log.Release("%s%s", p.who(), fmt.Sprintf(format, args...))
}

func (p *Agent) LogWarn(format string, args ...interface{}) {
	log.Warn("%s%s", p.who(), fmt.Sprintf(format, args...))
}

func (p *Agent) LogError(format string, args ...interface{}) {
	log.Error("%s%s", p.who(), fmt.Sprintf(format, args...))
}

func (p *Agent) OnClose(code uint) {
	SM.RunInSkeleton("gate.p.close", func() {
		p.LogRelease("player being destroyed:%d", code)
		p.Destroy()
	})
}

func (p *Agent) Addr() string {
	return p.conn.RemoteAddr().String()
}

func (p *Agent) Destroy() {
	if e := RoomMgr.Leave(p.fd, p.roomID); e != nil {
		p.LogWarn("player got no room")
	}
	p.conn.Close()
	p.destroyed = true
	DelAgentPlayer(p.fd)

	p.LogRelease("[%s] destroyed", p.conn.RemoteAddr())
}

func (p *Agent) readTask() {
	recvBuffer := new(bytes.Buffer)
	onceBuffer := make([]byte, 4096)
	for {
		msgHandler := func(msgID pb.CSMsgID, bodyBuf []byte) bool {
			reqBody := &pb.CSReqBody{}
			err := proto.Unmarshal(bodyBuf, reqBody)
			if err != nil {
				p.LogError("DealMsg %s Unmarshal fail[%s]", msgID, err.Error())
				return false
			}
			p.LogRelease(" ->Recv [%s][%s]", msgID, reqBody)
			SM.RPCServer.Go(msgID, p, msgID, reqBody)

			return true
		}

		if err := CSProcessor.DealMsgExt(p.conn, p, &p.encryptKey, recvBuffer, onceBuffer, msgHandler); err != nil {
			p.LogWarn("DealMsg failed[%s]", err.Error())
			break
		}
	}

	p.OnClose(uint(999))
}

func (p *Agent) update() {
	//不活跃踢线
	nowUnix := time.Now().Unix()
	if nowUnix-p.activeTime.Unix() > int64(conf.Server.PlayerInteractiveTime) {
		p.OnClose(1)
		p.LogWarn("inactive player [%s]", p.Addr())
		return
	}
}

func (p *Agent) SendClient(id pb.CSMsgID, message interface{}, onFinish func(error)) {
	safeFinish := func(e error) {
		if onFinish != nil {
			onFinish(e)
		} else {
			p.LogError("send msg failed e:%s", e.Error())
		}
	}

	if p.destroyed {
		safeFinish(errors.New("player is destroyed"))
		return
	}

	ret := RoomMgr.AddRoomTask(
		p.roomID,
		func() {
			if e := CSProcessor.WriteMsg(p.conn, id, message, &p.encryptKey); e != nil {
				p.LogWarn("send msg:%s failed:%s", id, e.Error())
			}
		}, nil,
	)

	if ret < 0 {
		p.LogError("add cs msg failed, ret:%d, msg:%s", ret, id)
	}
}
