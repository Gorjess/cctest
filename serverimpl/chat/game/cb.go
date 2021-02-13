package game

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/pb"
)

func reqLogin(p *Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.Login == nil {
		p.LogError("nil ReqLogin")
		return
	}

	rsp.Login = &pb.CSRspLogin{}

	roomID, e := RoomMgr.Join(p, req.Login.Username)
	if e != nil {
		rsp.ErrCode = pb.ERROR_CODE_FAILED
		rsp.ErrMsg = e.Error()
	} else {
		rsp.Login.RoomID = roomID
		rsp.Login.Username = req.Login.Username
	}
	p.SetUsername(req.Login.Username)

	p.SendClient(pb.CSMsgID_RSP_LOGIN, rsp, nil)
}

func reqHeartbeat(p *Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.Heartbeat == nil {
		p.LogError("nil Heartbeat")
		return
	}

	rsp.Heartbeat = &pb.CSRspHeartbeat{}
	p.SendClient(pb.CSMsgID_RSP_HEARTBEAT, rsp, nil)
}

func reqRoomChat(p *Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.RoomChat == nil {
		p.LogError("nil RoomChat")
		return
	}

	rsp.RoomChat = &pb.CSRspRoomChat{}
	RoomMgr.RoomChat(p.GetFD(), p.GetRoomID(), req.RoomChat.Content)
	p.SendClient(pb.CSMsgID_RSP_ROOM_CHAT, rsp, nil)
}

func reqRoomList(p *Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.RoomList == nil {
		p.LogError("nil RoomList")
		return
	}

	rsp.RoomList = &pb.CSRspRoomList{
		Rooms: RoomMgr.GetRoomList(req.RoomList.MaxRoomCount),
	}

	p.SendClient(pb.CSMsgID_RSP_ROOM_LIST, rsp, nil)
}

func reqJoinRoom(p *Agent, req *pb.CSReqBody, rsp *pb.CSRspBody) {
	if req.JoinRoom == nil {
		p.LogError("nil JoinRoom")
		return
	}

	log.Release("player:%s join room", p.username)

	rsp.JoinRoom = &pb.CSRspJoinRoom{}
	if _, e := RoomMgr.Join(p, p.GetUsername()); e != nil {
		rsp.ErrCode = pb.ERROR_CODE_FAILED
		rsp.ErrMsg = e.Error()
	}

	p.SendClient(pb.CSMsgID_RSP_JOIN_ROOM, rsp, nil)
}
