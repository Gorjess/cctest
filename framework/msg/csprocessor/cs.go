package csprocessor

import (
	"bytes"
	"cloudcadetest/common/compress/zlib"
	"cloudcadetest/common/encrypt/aes"
	"cloudcadetest/framework/agent"
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/module"
	"cloudcadetest/framework/network"
	"cloudcadetest/framework/network/protobuf"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

const HeadLenSize = 1

type CSMsgProcessor struct {
	*protobuf.Processor
	littleEndian    bool
	minMsgLen       uint32
	maxMsgLen       uint32
	minCompressSize uint32 //压缩阈值
	encrypt         bool   //是否加密
}

func New(
	srvMod *module.ServerMod,
	littleEndian bool,
	maxMsgLen, minCompressSize uint32,
	encrypt bool) *CSMsgProcessor {
	p := &CSMsgProcessor{
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

func (p *CSMsgProcessor) SetMinCompressSize(minCompressSize uint32) {
	atomic.StoreUint32(&p.minCompressSize, minCompressSize)
}

func (p *CSMsgProcessor) GetMinCompressSize() uint32 {
	return atomic.LoadUint32(&p.minCompressSize)
}

func (p *CSMsgProcessor) DealMsgExt(conn network.IConn, agent agent.Agent, key *aes.Key,
	recvBuffer *bytes.Buffer, onceBuffer []byte, msgHandler func(CSMsgID, []byte) bool) error {

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
		hlen := uint32(buf[0])
		if hlen > p.maxMsgLen {
			return fmt.Errorf("message too long %d", hlen)
		} else if hlen <= p.minMsgLen {
			return fmt.Errorf("message too short %d", hlen)
		}

		if rlen < int(hlen)+HeadLenSize {
			break
		}

		//parse head
		h := &CSHead{}
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

			if h.Compress {
				//需要解压缩
				if bodyBuf, err = zlib.DoZlibUnCompress(bodyBuf); err != nil {
					return errors.New("DoZlibUnCompress fail:" + err.Error())
				}
			}
		}

		if msgHandler != nil {
			if !msgHandler(h.MsgId, bodyBuf) {
				return errors.New("msg handler err")
			}
		}
	}
	return nil
}

func (p *CSMsgProcessor) GetCompressData(bodyData []byte) ([]byte, bool) {
	compress := false
	bodyLen := uint32(len(bodyData))

	if bodyLen > 0 {
		minCompressSize := p.GetMinCompressSize()
		if minCompressSize != 0 && bodyLen >= minCompressSize {
			//需要进行压缩
			now := time.Now()
			zipData, err := zlib.Compress(bodyData)
			dt := time.Since(now)
			if dt > 5*time.Millisecond {
				// 统计压缩时间，对于超时的记录下基础信息
				log.Warn("zlib.DoZlibCompress timeout: %d(len), %d(dt)", len(bodyData), dt/1e6)
			}
			zipLen := uint32(len(zipData))
			if err == nil && zipLen < bodyLen {
				bodyData = zipData
				compress = true
			}
		}
	}

	return bodyData, compress
}

func (p *CSMsgProcessor) encryptMsg(id CSMsgID, bodyData []byte, key *aes.Key) ([]byte, error) {
	if p.encrypt && len(bodyData) > 0 {
		if key == nil {
			return nil, errors.New("no Encrypt key")
		}
		now := time.Now()
		cryptData, err := aes.Encrypt(bodyData, key.K)
		dt := time.Since(now)
		if dt > 5*time.Millisecond { // 统计下事件
			log.Release("aes.Encrypt timeout %v %v %v", id, len(bodyData), dt)
		}

		if err != nil {
			return nil, errors.New("Encrypt fail:" + err.Error())
		}

		bodyData = cryptData
	}

	return bodyData, nil
}

func (p *CSMsgProcessor) doWriteData(conn network.IConn, id CSMsgID, encryptedData []byte, compress bool) error {
	bodyLen := uint32(len(encryptedData))

	h := &CSHead{
		MsgId:    id,
		BodyLen:  bodyLen,
		Compress: compress,
	}

	defer func() {
		atomic.AddInt32(&p.stat.SendMsgCount, 1)
		atomic.AddInt32(&p.stat.SendBytes, int32(bodyLen))
	}()
	headData, err := p.Processor.Marshal([]interface{}{int32(id), h})
	if err != nil {
		return errors.New("Marshal head error:" + err.Error())
	}
	headLen := uint32(len(headData))

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
			binary.LittleEndian.PutUint32(data, headLen)
		} else {
			binary.BigEndian.PutUint32(data, headLen)
		}
	}

	copy(data[HeadLenSize:], headData)
	if bodyLen != 0 {
		copy(data[HeadLenSize+headLen:], encryptedData)
	}

	return conn.Write(data)
}

func (p *CSMsgProcessor) CompressMsg(id CSMsgID, message interface{}) ([]byte, bool, error) {
	body, er := p.marshalMsg(id, message)
	if er != nil {
		return nil, false, er
	}

	var (
		lenBefore    = len(body) // 计算压缩比用
		lenAfter     int
		isCompressed bool
	)

	body, isCompressed = p.GetCompressData(body)
	lenAfter = len(body)

	if isCompressed {
		log.Release("compressed msg, id:%s, before:%d, after:%d, ratio:%f",
			id, lenBefore, lenAfter, float32(lenAfter)/float32(lenBefore))
	}

	return body, isCompressed, nil
}

func (p *CSMsgProcessor) Write2Socket(conn network.IConn, id CSMsgID, byteMsg []byte, isCompressed bool, key *aes.Key) error {
	data, er := p.encryptMsg(id, byteMsg, key)
	if er != nil {
		return er
	}

	return p.doWriteData(conn, id, data, isCompressed)
}

func (p *CSMsgProcessor) doWriteBodyData(conn network.IConn, id CSMsgID, bodyData []byte, key *aes.Key) error {
	var (
		bodyLen       uint32
		compress      bool
		bodyLenBefore = uint32(len(bodyData))
	)

	if len(bodyData) > 0 {
		bodyData, compress = p.GetCompressData(bodyData)
		bodyLen = uint32(len(bodyData))
		if compress {
			log.Release("compressed msg, id:%s, before:%d, after:%d, ratio:%f",
				id, bodyLenBefore, bodyLen, float32(bodyLen)/float32(bodyLenBefore))
		}

		if p.encrypt && bodyLen > 0 {
			//需要加密
			if key == nil {
				return errors.New("no Encrypt key")
			}
			now := time.Now()
			cryptData, err := aes.Encrypt(bodyData, key.K)
			dt := time.Since(now)
			if dt > 5*time.Millisecond { // 统计下事件
				log.Release("aes.Encrypt timeout %v %v %v", id, len(bodyData), dt)
			}

			if err != nil {
				return errors.New("Encrypt fail:" + err.Error())
			}

			bodyData = cryptData
			bodyLen = uint32(len(bodyData))
		}
	}

	return p.doWriteData(conn, id, bodyData, compress)
}

func (p *CSMsgProcessor) WriteMsg(conn *network.TCPConn, id CSMsgID, msg interface{}, key *aes.Key) error {
	copyKey := *key
	er := p.doWriteMsg(conn, id, msg, &copyKey)
	if er != nil {
		p.doPrintWriteError(conn, int(id), er)
		return er
	}

	return nil
}

func (p *CSMsgProcessor) marshalMsg(id CSMsgID, msg interface{}) ([]byte, error) {
	if msg == nil {
		return nil, nil
	}
	now := time.Now()
	bodyData, e := p.Processor.Marshal([]interface{}{int32(id), msg})
	if e != nil {
		err = errors.New(fmt.Sprintf("Marshal id:%v error:%v", id, error.Error()))
		return
	}

	dt := time.Since(now)
	if dt > 5*time.Millisecond { // 大于5 ms的要统计下
		log.Release("msg marshal timeout msgId=%v len=%v dt=%v", id, len(bodyData), dt)
	}
	return
}

func (p *CSMsgProcessor) doWriteMsg(conn network.IConn, id CSMsgID, msg interface{}, key *aes.Key) error {
	//log.Debug("doWriteMsg %v %v %v %v", conn, id, msg, key)
	bodyData, err := p.marshalMsg(id, msg)
	if err != nil {
		return err
	}
	return p.doWriteBodyData(conn, id, bodyData, key)
}

func (p *CSMsgProcessor) NeedEncrypt() bool {
	return p.encrypt
}

func (p *CSMsgProcessor) Stop() {
	p.task.Stop()
}

func (p *CSMsgProcessor) GetTaskLen() int {
	return p.task.Len()
}

func (p *CSMsgProcessor) CloseConn(conn network.IConn) {
	p.task.AddTask(func() {
		if conn != nil {
			conn.Close()
		}
	}, nil)
}

func (p *CSMsgProcessor) fiterError(msgIdNumber int, errorMsg string) bool {
	switch errorMsg {
	/*	case "write channel full":
		log.Warn("msg %v WriteBodyData err %v", msgIdNumber,errorMsg)
		fallthrough*/
	case "write channel is closed":
		return true
	}
	return false
}

func (p *CSMsgProcessor) doPrintWriteError(conn network.IConn, msgIdNumber int, er error) {
	if er == nil || conn == nil {
		return
	}
	erMsg := er.Error()
	if erMsg == "write channel is closed" {
		return
	}
	writeStat := ""
	if erMsg == "write channel full" {
		if conn.Stat() != nil && conn.Stat().WriteStat != nil {
			writeStat = conn.Stat().WriteStat.Print()
		}
		log.Warn("ip[%v] msg[%v] WriteBodyData err[%v] writestat[%v]", conn.RemoteAddr(), msgIdNumber, er, writeStat)
	} else {
		log.Error("ip[%v] msg[%v] WriteBodyData err[%v]", conn.RemoteAddr(), msgIdNumber, er)
	}
}
