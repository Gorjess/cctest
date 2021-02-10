package uuid

import "time"

// 通过时间戳+(该秒内自增id)
type UUID struct {
	high int64 // 24~64位(可表示未来无数年 目前1970到现在的时间戳大概在 16亿左右 40位来表示足够了)
	low  int32 // 自增id 0~23 位 2^24 1s内的自增id 足够了
}

func (uuid *UUID) reset() {
	now := time.Now().Unix()
	if uuid.high == now { // 同一时刻不更换
		return
	}
	uuid.high = now
	uuid.low = 0
}

func (uuid *UUID) Get() int64 {
	uuid.reset()
	uuid.low++
	return (uuid.high << 24) + int64(uuid.low)
}
