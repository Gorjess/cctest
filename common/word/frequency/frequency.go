package frequency

import (
	"cloudcadetest/common/ostype"
	"cloudcadetest/common/word/frequency/wordmeta"
	"cloudcadetest/common/word/frequency/wordsbysec"
	"cloudcadetest/framework/log"
	"container/list"
	"sort"
	"strings"
	"time"
)

type sortTask struct {
	lastNSeconds int
	cb           func(meta *wordmeta.Data, e error)
}

type Frequency struct {
	heapNodeCount int
	frqBySec      *list.List
	wordChan      chan string
	sortTasks     chan *sortTask
}

func New() *Frequency {
	f := &Frequency{
		frqBySec:  list.New(),
		wordChan:  make(chan string, 100),
		sortTasks: make(chan *sortTask, 100),
	}
	go f.update()

	return f
}

func (f *Frequency) update() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			f.onTimePassed()

		case word := <-f.wordChan:
			// consider tail as the node for the current second
			tail := f.frqBySec.Back()
			if tail != nil {
				ws, ok := tail.Value.(*wordsbysec.Words)
				if !ok {
					goto END
				}
				ws.Add(word)
			} else {
				goto END
			}

		case task := <-f.sortTasks:
			f.processTask(task)
		}
	}

END:
	log.Error("frequency handler quits")
}

func (f *Frequency) onTimePassed() {
	if f.frqBySec.Len() == 60 {
		h := f.frqBySec.Front()
		f.frqBySec.Remove(h)
	}
	f.frqBySec.PushBack(wordsbysec.New(1000))
}

func (f *Frequency) Add(sentence string) {
	cutset := "\n"
	if ostype.Get() == ostype.Windows {
		cutset = "\r\n"
	}

	words := strings.Split(strings.TrimRight(sentence, cutset), " ")
	for _, w := range words {
		select {
		case f.wordChan <- w:
		default:
			log.Error("too many pending words, lost:%s", w)
		}
	}
}

func (f *Frequency) GetFrequencyByTime(lastNSeconds int, cb func(meta *wordmeta.Data, e error)) {
	if lastNSeconds > 60 || lastNSeconds < 1 || cb == nil {
		return
	}

	select {
	case f.sortTasks <- &sortTask{
		lastNSeconds: lastNSeconds,
		cb:           cb,
	}:
	default:
		log.Error("too many pending get-freq tasks, lost:%d", lastNSeconds)
	}
}

func (f *Frequency) processTask(task *sortTask) {
	if task == nil {
		return
	}

	var (
		frqs  = make(wordmeta.Datas, 0, task.lastNSeconds)
		node  *list.Element
		wmeta *wordmeta.Data
		ok    bool
	)

	for i := 0; i < task.lastNSeconds; i++ {
		node = f.frqBySec.Back()
		if node == nil || node.Value == nil {
			break
		}
		wmeta, ok = node.Value.(*wordmeta.Data)
		if !ok {
			break
		}
		frqs = append(frqs, wmeta)
	}

	sort.Sort(frqs)
	task.cb(frqs[0], nil)
}
