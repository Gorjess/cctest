package msghandler

import (
	"cloudcadetest/pb"
	"cloudcadetest/serverimpl/chat/game"
	"cloudcadetest/serverimpl/chat/game/agentmgr/player"
)

func reqLogin(p *player.Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.Login == nil {
		p.LogError("nil ReqLogin")
		return
	}

	rsp.Login = &pb.CSRspLogin{}

	id, e := game.RoomMgr.Join(p.GetFD())
	if e != nil {
		rsp.ErrCode = pb.ERROR_CODE_FAILED
		rsp.ErrMsg = e.Error()
	} else {
		p.SetRoomID(id)
	}
	p.SendClient(pb.CSMsgID_RSP_LOGIN, rsp, nil)
}

func reqHeartbeat(p *player.Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.Heartbeat == nil {
		p.LogError("nil Heartbeat")
	}

	rsp.Heartbeat = &pb.CSRspHeartbeat{}
	p.SendClient(pb.CSMsgID_RSP_HEARTBEAT, rsp, nil)
}