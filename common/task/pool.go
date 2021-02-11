package task

import (
	"cloudcadetest/framework/log"
	"cloudcadetest/framework/module"
	"hash/fnv"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
)

type taskFuncPair struct {
	f  func()
	cb func()
}

// 更新任务
type UpdateTask struct {
	t  chan *taskFuncPair
	sm *module.ServerMod
}

type Pool struct {
	Tasks    []*UpdateTask
	curIndex uint32
	chanNum  int
	stoped   bool

	rndIdx int
	name   string
}

// 仅用于在压测时计算吞吐量
// 生产环境中不要读写这两个值
var (
	taskProduced = uint32(0)
	taskConsumed = uint32(0)
)

func NewTaskPool(sm *module.ServerMod, taskNum, chanNum int) *Pool {
	pool := &Pool{
		Tasks: []*UpdateTask{},
	}

	//根据配置初始化, 否则设置默认值
	if taskNum <= 0 {
		taskNum = 300
	}
	if chanNum <= 0 {
		chanNum = 10000
	}

	pool.chanNum = chanNum
	for i := 0; i < taskNum; i++ {
		task := &UpdateTask{
			t:  make(chan *taskFuncPair, chanNum),
			sm: sm,
		}

		pool.Tasks = append(pool.Tasks, task)

		go ProcessTask(task)
	}

	return pool
}

func (p *Pool) Start() {

}

func (p *Pool) Stop() {
	if p.stoped {
		return
	}

	p.stoped = true

	for _, task := range p.Tasks {
		close(task.t)
	}
}

// 固定到指定的pool上
func (p *Pool) AddTask(f func(), cb func(), poolDecide string) {
	if len(p.Tasks) == 0 {
		log.Error("pool task is 0")
		return
	}
	var index uint32

	chanAllFull := false
	if poolDecide == "" {
		index = p.curIndex

		// 从当前序号开始找一个未满的task
		for {
			if len(p.Tasks[int(index)].t) < p.chanNum {
				break
			}
			index = (index + 1) % uint32(len(p.Tasks))

			// 当轮询所有task都已经满了后 返回
			if index == p.curIndex {
				chanAllFull = true
				break
			}
		}

		// 指向下一个task序号
		p.curIndex = (index + 1) % uint32(len(p.Tasks))
	} else {
		// 玩家id固定到对应的task上 保证先后
		index = HashString(poolDecide) % uint32(len(p.Tasks))
	}

	t := p.Tasks[int(index)].t
	if len(t) >= p.chanNum {
		_, file, line, _ := runtime.Caller(1)
		ss := strings.Split(file, "/")
		fileName := ss[len(ss)-1]

		id := "[" + fileName + ":" + strconv.Itoa(line) + "] "

		if chanAllFull {
			log.Warn("add task[%v]. all task is full", id)
		} else {
			log.Warn("add task[%v]. taskPool index:%d is full", id, index)
		}

	}

	select {
	case t <- &taskFuncPair{
		f:  f,
		cb: cb,
	}:
		atomic.AddUint32(&taskProduced, 1)
		produced := atomic.LoadUint32(&taskProduced)
		if produced%1000 == 0 {
			log.Warn("tp task produced:%d", produced)
		}

	default:
		log.Error("task is full")
	}
}

func (t *UpdateTask) executeFun(pair *taskFuncPair) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			log.Error("%v: %s", r, buf[:l])
		}
	}()

	if pair.f != nil {
		pair.f()
	}

	if pair.cb != nil {
		t.sm.RunInSkeleton("task.cb", pair.cb)
	}
}

func ProcessTask(task *UpdateTask) {
	if task == nil {
		panic("task is nil")
		return
	}

	for {
		pair, ok := <-task.t
		if !ok {
			break
		}

		atomic.AddUint32(&taskConsumed, 1)
		consumed := atomic.LoadUint32(&taskConsumed)
		if consumed%1000 == 0 {
			log.Warn("tp task consumed:%d", consumed)
		}

		task.executeFun(pair)
	}
}

func HashString(s string) uint32 {
	h := fnv.New32a()
	if _, er := h.Write([]byte(s)); er != nil {
		log.Error("write hash:%s failed:%s", s, er.Error())
	}
	return h.Sum32()
}

func (p *Pool) Len() int {
	num := 0
	for _, task := range p.Tasks {
		if task == nil {
			continue
		}

		num += len(task.t)
	}

	return num
}

//--========================== round distribution ==========================--
func (p *Pool) SetName(name string) *Pool {
	if p != nil {
		p.name = name
	}
	return p
}

func (p *Pool) AddFixedTask(f, cb func(), idx int) int {
	if len(p.Tasks) == 0 {
		return -1
	}

	if idx < 0 || idx >= len(p.Tasks) {
		idx = (p.rndIdx + 1) % len(p.Tasks)
		p.rndIdx = idx
	}

	select {
	case p.Tasks[idx].t <- &taskFuncPair{f, cb}:
	default:
		log.Error("task pool[%s]'s sub chan[idx=%d] full.", p.name, idx)
		idx = -1
	}

	return idx
}
