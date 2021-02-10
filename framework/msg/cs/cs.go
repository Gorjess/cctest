package cs

import (
	"bytes"
	"cloudcadetest/common/compress/zlib"
	"cloudcadetest/common/encrypt/aes"
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/module"
	"cloudcadetest/framework/network"
	"cloudcadetest/framework/network/protobuf"
	"cloudcadetest/pb"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

const HeadLenSize = 1

type Processor struct {
	*protobuf.Processor
	littleEndian    bool
	minMsgLen       int32
	maxMsgLen       int32
	minCompressSize int32 //压缩阈值
	encrypt         bool  //是否加密
}

func New(
	srvMod *module.ServerMod,
	littleEndian bool,
	maxMsgLen, minCompressSize int32,
	encrypt bool) *Processor {
	p := &Processor{
		Processor:    protobuf.NewProcessor(),
		littleEndian: littleEndian,
		minMsgLen:    0,
		maxMsgLen:    maxMsgLen,
		encrypt:      encrypt,
	}

	p.SetMinCompressSize(minCompressSize)

	time.Since(time.Now()).Nanoseconds()
	p.Processor.SetRouter(srvMod.RPCServer)

	return p
}

func (p *Processor) SetMinCompressSize(minCompressSize int32) {
	atomic.StoreInt32(&p.minCompressSize, minCompressSize)
}

func (p *Processor) GetMinCompressSize() int32 {
	return atomic.LoadInt32(&p.minCompressSize)
}

func (p *Processor) DealMsgExt(conn network.IConn, agent agent.Agent, key *aes.Key,
	recvBuffer *bytes.Buffer, onceBuffer []byte, msgHandler func(pb.CSMsgID, []byte) bool) error {

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
		if rlen < HeadLenSize { // 包不够长度
			break
		}

		buf := recvBuffer.Bytes()
		hlen := int32(buf[0])
		if hlen > p.maxMsgLen {
			return fmt.Errorf("message too long %d", hlen)
		} else if hlen <= p.minMsgLen {
			return fmt.Errorf("message too short %d", hlen)
		}

		if rlen < int(hlen)+HeadLenSize {
			break
		}

		//parse head
		h := &pb.CSHead{}
		if err = p.Processor.Unmarshal(buf[HeadLenSize:hlen+HeadLenSize], h); err != nil {
			return errors.New("Unmarshal head error:" + err.Error() + fmt.Sprintf(" headLen:%v", hlen))
		}

		if h.BodyLen > p.maxMsgLen {
			return fmt.Errorf("message too long %v", h.BodyLen)
		} else if h.BodyLen < p.minMsgLen {
			return fmt.Errorf("message too short %v", h.BodyLen)
		}

		pktLen := int(hlen) + HeadLenSize + int(h.BodyLen)
		if rlen < pktLen {
			break
		}

		data := recvBuffer.Next(pktLen)
		var bodyBuf []byte
		if h.BodyLen > 0 {
			bodyBuf = make([]byte, h.BodyLen, h.BodyLen)
			if copy(bodyBuf, data[int(hlen)+HeadLenSize:pktLen]) != int(h.BodyLen) { // 拷贝出错了
				log.Release("copy err:%v", h)
				return errors.New("copy err")
			}

			if p.encrypt {
				//需要解密
				if key == nil {
					return errors.New("no Encrypt key")
				}

				if bodyBuf, err = aes.Decrypt(bodyBuf, key.K); err != nil {
					return errors.New("Decrypt fail:" + err.Error())
				}
			}

			if h.IsCompressed {
				//需要解压缩
				if bodyBuf, err = zlib.Compress(bodyBuf); err != nil {
					return errors.New("DoZlibUnCompress fail:" + err.Error())
				}
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

func (p *Processor) GetCompressData(bodyData []byte) ([]byte, bool) {
	compress := false
	bodyLen := int32(len(bodyData))

	if bodyLen > 0 {
		minCompressSize := p.GetMinCompressSize()
		if minCompressSize != 0 && bodyLen >= minCompressSize {
			//需要进行压缩
			now := time.Now()
			zipData, err := zlib.Compress(bodyData)
			dt := time.Since(now)
			if dt > 5*time.Millisecond {
				// 统计压缩时间，对于超时的记录下基础信息
				log.Warn("zlib.Compress timeout: %d(len), %d(dt)", len(bodyData), dt/1e6)
			}
			zipLen := int32(len(zipData))
			if err == nil && zipLen < bodyLen {
				bodyData = zipData
				compress = true
			}
		}
	}

	return bodyData, compress
}

func (p *Processor) encryptMsg(id pb.CSMsgID, bodyData []byte, key *aes.Key) ([]byte, error) {
	if p.encrypt && len(bodyData) > 0 {
		if key == nil {
			return nil, errors.New("no Encrypt key")
		}
		now := time.Now()
		cryptData, err := aes.Encrypt(bodyData, key.K)
		dt := time.Since(now)
		if dt > 5*time.Millisecond {
			log.Release("aes.Encrypt timeout %v %v %v", id, len(bodyData), dt)
		}

		if err != nil {
			return nil, errors.New("Encrypt fail:" + err.Error())
		}

		bodyData = cryptData
	}

	return bodyData, nil
}

func (p *Processor) doWriteData(conn network.IConn, id pb.CSMsgID, encryptedData []byte, compress bool) error {
	bodyLen := int32(len(encryptedData))

	h := &pb.CSHead{
		MsgID:        id,
		BodyLen:      bodyLen,
		IsCompressed: compress,
	}

	headData, err := p.Processor.Marshal([]interface{}{int32(id), h})
	if err != nil {
		return errors.New("Marshal head error:" + err.Error())
	}
	headLen := int32(len(headData))

	data := make([]byte, HeadLenSize+headLen+bodyLen)
	// write headlen
	switch HeadLenSize {
	case 1:
		data[0] = byte(headLen)
	case 2:
		if p.littleEndian {
			binary.LittleEndian.PutUint16(data, uint16(headLen))
		} else {
			binary.BigEndian.PutUint16(data, uint16(headLen))
		}
	case 4:
		if p.littleEndian {
			binary.LittleEndian.PutUint32(data, uint32(headLen))
		} else {
			binary.BigEndian.PutUint32(data, uint32(headLen))
		}
	}

	copy(data[HeadLenSize:], headData)
	if bodyLen != 0 {
		copy(data[HeadLenSize+headLen:], encryptedData)
	}

	return conn.Write(data)
}

func (p *Processor) CompressMsg(id pb.CSMsgID, message interface{}) ([]byte, bool, error) {
	body, er := p.marshalMsg(id, message)
	if er != nil {
		return nil, false, er
	}

	var isCompressed bool
	body, isCompressed = p.GetCompressData(body)

	return body, isCompressed, nil
}

func (p *Processor) Write2Socket(conn network.IConn, id pb.CSMsgID, byteMsg []byte, isCompressed bool, key *aes.Key) error {
	data, er := p.encryptMsg(id, byteMsg, key)
	if er != nil {
		return er
	}

	return p.doWriteData(conn, id, data, isCompressed)
}

func (p *Processor) doWriteBodyData(conn network.IConn, id pb.CSMsgID, bodyData []byte, key *aes.Key) error {

	var isCompressed bool
	if len(bodyData) > 0 {
		bodyData, isCompressed = p.GetCompressData(bodyData)

		if p.encrypt && len(bodyData) > 0 {
			//需要加密
			if key == nil {
				return errors.New("no Encrypt key")
			}
			now := time.Now()
			cryptData, err := aes.Encrypt(bodyData, key.K)
			dt := time.Since(now)
			if dt > 5*time.Millisecond {
				log.Release("aes.Encrypt timeout %s %d %s", id, len(bodyData), dt)
			}

			if err != nil {
				return errors.New("Encrypt fail:" + err.Error())
			}

			bodyData = cryptData
		}
	}

	return p.doWriteData(conn, id, bodyData, isCompressed)
}

func (p *Processor) WriteMsg(conn network.IConn, id pb.CSMsgID, msg interface{}, key *aes.Key) error {
	copyKey := *key
	er := p.doWriteMsg(conn, id, msg, &copyKey)
	if er != nil {
		return er
	}

	return nil
}

func (p *Processor) marshalMsg(id pb.CSMsgID, msg interface{}) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	now := time.Now()
	bodyData, e := p.Processor.Marshal([]interface{}{int32(id), msg})
	if e != nil {
		e = errors.New(fmt.Sprintf("Marshal id:%s error:%v", id, e.Error()))
		return nil, e
	}

	dt := time.Since(now)
	if dt > 5*time.Millisecond {
		log.Release("msg marshal timeout msgID=%s len=%d dt=%s", id, len(bodyData), dt)
	}
	return bodyData, nil
}

func (p *Processor) doWriteMsg(conn network.IConn, id pb.CSMsgID, msg interface{}, key *aes.Key) error {
	bodyData, err := p.marshalMsg(id, msg)
	if err != nil {
		return err
	}
	return p.doWriteBodyData(conn, id, bodyData, key)
}

func (p *Processor) NeedEncrypt() bool {
	return p.encrypt
}

func (p *Processor) Stop() {
}

func (p *Processor) CloseConn(conn network.IConn) {
	if conn != nil {
		conn.Close()
	}
}
