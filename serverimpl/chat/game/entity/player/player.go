package player

import (
	"bytes"
	"cloudcadetest/common/encrypt/aes"
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/network"
	"cloudcadetest/pb"
	"cloudcadetest/serverimpl/chat/conf"
	"cloudcadetest/serverimpl/chat/game"
	"cloudcadetest/serverimpl/chat/game/entity"
	"errors"
	"google.golang.org/protobuf/proto"
	"sync/atomic"
	"time"
)

type Player struct {
	conn   network.IConn //玩家连接
	FD     int64
	roomID int64

	activeTime int64 //活跃时间
	destroyed  bool  //已销毁标志
	encryptKey aes.Key
	login      bool
	working    bool //标识连接状态(false 等待客户端发送第一个包 true 收到客户端第一个包后进入工作模式)
}

func New(conn *network.TCPConn) agent.Agent {
	if conn == nil {
		log.Error("conn is nil")
		return nil
	}

	p := &Player{
		conn: conn,
	}

	atomic.StoreInt64(&p.activeTime, time.Now().Unix())

	p.FD = game.UUID.Get()

	game.Server.GetEntity().RunInSkeleton("gate.new.agent", func() {
		entity.AddAgentPlayer(p)
	})

	log.Release("player[%s][%d] connected", p.conn.RemoteAddr(), p.FD)

	//处理读任务
	p.readTask()

	return p
}

func (p *Player) OnClose(code uint) {
	game.Server.GetEntity().RunInSkeleton("gate.p.close", func() {
		p.Destroy()
	})
}

func (p *Player) Addr() string {
	return p.conn.RemoteAddr().String()
}

func (p *Player) Destroy() {
	//写关闭
	p.conn.Close()

	p.destroyed = true
	entity.DelAgentPlayer(p.FD)

	log.Release("player[%s][%d] destroyed", p.conn.RemoteAddr(), p.FD)
}

func (p *Player) readTask() {
	recvBuffer := new(bytes.Buffer)
	onceBuffer := make([]byte, 4096)
	for {
		msgHandler := func(msgID pb.CSMsgID, bodyBuf []byte) bool {
			reqBody := &pb.CSReqBody{}
			err := proto.Unmarshal(bodyBuf, reqBody)
			if err != nil {
				log.Error("player[%d] DealMsg %s Unmarshal fail[%s]", p.FD, msgID, err.Error())
				return false
			}
			game.Server.GetEntity().RPCServer.Go(msgID, p, msgID, reqBody)

			return true
		}

		if err := game.CSProcessor.DealMsgExt(p.conn, p, &p.encryptKey, recvBuffer, onceBuffer, msgHandler); err != nil {
			log.Warn("player[%d] DealMsg failed[%s]", p.FD, err.Error())
			break
		}
	}

	p.OnClose(uint(999))
}

func (p *Player) update() {
	//不活跃踢线
	nowUnix := time.Now().Unix()
	if nowUnix-atomic.LoadInt64(&p.activeTime) > int64(conf.Server.PlayerInteractiveTime) {
		p.OnClose(1)
		log.Warn("inactive player [%s][%d]", p.Addr(), p.FD)
		return
	}
}

func (p *Player) sendClient(id pb.CSMsgID, message interface{}, onFinish func(error)) {
	safeFinish := func(e error) {
		if onFinish != nil {
			onFinish(e)
		} else {
			log.Error("send msg failed, p:%d, room:%d, e:%s",
				p.FD, p.roomID, e.Error())
		}
	}

	if p.destroyed {
		safeFinish(errors.New("player is destroyed"))
		return
	}

	ret := game.RoomMgr.AddRoomTask(
		p.roomID,
		func() {
			// 已经在函数内打印了错误信息
			// 这里不再处理这个error
			if e := game.CSProcessor.WriteMsg(p.conn, id, message, &p.encryptKey); e != nil {
				log.Warn("send msg:%s failed:%s, p:%d, room:%d", id, e.Error(), p.FD, p.roomID)
			}
		}, nil,
	)

	if ret < 0 {
		log.Error("add cs msg failed, ret:%d, msg:%s, p:%d, room:%d", ret, id, p.FD, p.roomID)
	}
}
