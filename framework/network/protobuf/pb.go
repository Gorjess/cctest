package protobuf

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/rpc"
	"google.golang.org/protobuf/proto"

	"errors"
	"time"
)

// | id | protobuf message |
type Processor struct {
	router *rpc.Server
}

func NewProcessor() *Processor {
	p := new(Processor)
	return p
}

// Do not call SetRouter on routing or (un)marshalling
func (p *Processor) SetRouter(msgRouter *rpc.Server) {
	p.router = msgRouter
}

// goroutine safe
func (p *Processor) Marshal(msg interface{}) ([]byte, error) {
	var (
		start = time.Now()
		conv  proto.Message
		msgID int32
	)

	switch args := msg.(type) {
	case []interface{}:
		msgID = args[0].(int32)
		conv = args[1].(proto.Message)
	case proto.Message:
		conv = args
	default:
		return nil, errors.New("unknown msg type")
	}

	convertDur := time.Since(start)
	if convertDur > time.Millisecond*50 {
		log.Release("myproto.Marshal.conv dur:%s, msgID:%d, msg:%v", convertDur, msgID, msg)
	}

	data, err := proto.Marshal(conv)
	if err != nil {
		return nil, err
	}
	marshalDur := time.Since(start)
	if marshalDur > time.Millisecond*100 {
		log.Release("myproto.Marshal convertDur:%s, marshalDur:%s, msgID:%d, datalen:%d", convertDur, msgID, len(data))
	}

	return data, nil
}

// goroutine safe
func (p *Processor) Unmarshal(data []byte, msg interface{}) error {
	return proto.Unmarshal(data, msg.(proto.Message))
}

// goroutine safe
func (p *Processor) Route(msgId interface{}, args ...interface{}) error {
	if p.router != nil {
		p.router.Go(msgId, args...)
	}
	return nil
}
