package msghandler

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/pb"
	"cloudcadetest/serverimpl/chat/game"
	"cloudcadetest/serverimpl/chat/game/agentmgr/player"
	"time"
)

var (
	functions map[pb.CSMsgID]func(*player.Agent, *pb.CSReqBody, *pb.CSRspBody)
)

func handlerCS(repId pb.CSMsgID, f func(*player.Agent, *pb.CSReqBody, *pb.CSRspBody)) {
	if !game.Server.GetEntity().IsRegister(repId) {
		game.Server.GetEntity().RegisterChanRPC(repId, handleCS)
	}
	functions[repId] = f
}

func checkPlayer(arg interface{}) *player.Agent {
	p, ok := arg.(*player.Agent)
	if !ok {
		log.Warn("invalid find player")
		return nil
	}

	if p.IsDestroyed() {
		return nil
	}

	return p
}

func handleCS(args []interface{}) {
	//player
	p := checkPlayer(args[0])
	if p == nil {
		return
	}

	//p.msgAccumulateCount++

	// id
	reqId, ok := args[1].(pb.CSMsgID)
	if !ok {
		log.Error("invalid find req id")
		return
	}

	// Req 消息
	req, ok := args[2].(*pb.CSReqBody)
	if !ok {
		log.Error("invalid req msg")
		return
	}

	//处理函数
	f, ok := functions[reqId]
	if !ok {
		log.Error("cb not found, player[%d] reqId[%s]", p.GetFD(), reqId)
		return
	}

	//执行
	rsp := &pb.CSRspBody{
		Seq: req.Seq,
	}
	f(p, req, rsp)

	p.UpdateActiveTime(time.Now())
}

func registerHandler() {
	functions = map[pb.CSMsgID]func(*player.Agent, *pb.CSReqBody, *pb.CSRspBody){}

	handlerCS(pb.CSMsgID_REQ_LOGIN, reqLogin)
	handlerCS(pb.CSMsgID_REQ_HEARTBEAT, reqHeartbeat)
}

func reply(p *player.Agent, id pb.CSMsgID, rsp *pb.CSRspBody, err error) {
	if p == nil {
		log.Error("invalid player")
		return
	}

	if err != nil && rsp != nil {
		rsp.ErrCode = pb.ERROR_CODE_FAILED
		rsp.ErrMsg = err.Error()
		log.Warn("player[%d] %v", p.GetFD(), err.Error())
	}

	if rsp != nil {
		game.RoomMgr.AddRoomTask(p.GetRoomID(), func() {
			if e := game.CSProcessor.WriteMsg(p.GetConn(), id, rsp, p.GetEncKey()); e != nil {
				log.Warn("write reply:%s failed:%s, p:%d", id, e.Error(), p.GetFD())
			}
		}, nil)
	}
}
