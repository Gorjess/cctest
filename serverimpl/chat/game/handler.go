package game

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/pb"
	"time"
)

var (
	functions map[pb.CSMsgID]func(*Agent, *pb.CSReqBody, *pb.CSRspBody)
)

func handlerCS(repId pb.CSMsgID, f func(*Agent, *pb.CSReqBody, *pb.CSRspBody)) {
	if !SM.IsRegister(repId) {
		SM.RegisterChanRPC(repId, handleCS)
	}
	functions[repId] = f
}

func checkPlayer(arg interface{}) *Agent {
	p, ok := arg.(*Agent)
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
	functions = map[pb.CSMsgID]func(*Agent, *pb.CSReqBody, *pb.CSRspBody){}

	handlerCS(pb.CSMsgID_REQ_LOGIN, reqLogin)
	handlerCS(pb.CSMsgID_REQ_HEARTBEAT, reqHeartbeat)
	handlerCS(pb.CSMsgID_REQ_ROOM_CHAT, reqRoomChat)
	handlerCS(pb.CSMsgID_REQ_ROOM_LIST, reqRoomList)
	handlerCS(pb.CSMsgID_REQ_JOIN_ROOM, reqJoinRoom)
}
