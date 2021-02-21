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
	ts           int64
	lastNSeconds int
	cb           func(meta *wordmeta.Data, e error)
}

type Frequency struct {
	heapNodeCount int
	frqBySec      *list.List
	wordChan      chan string
	sortTasks     chan *sortTask
	produced      int32
	consumed      int32
}

func New() *Frequency {
	f := &Frequency{
		frqBySec:  list.New(),
		wordChan:  make(chan string, 3500),
		sortTasks: make(chan *sortTask, 10000),
	}
	f.frqBySec.PushBack(newSecWords())
	go f.update()

	return f
}

func newSecWords() *wordsbysec.Words {
	return wordsbysec.New(1000)
}

func (f *Frequency) update() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			f.onTimePassed()

		case word := <-f.wordChan:
			// use tail as the node for the current second
			//fmt.Printf("%d consume word:%s\n", time.Now().UnixNano(), word)
			//atomic.AddInt32(&f.consumed, 1)
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
	f.frqBySec.PushBack(newSecWords())
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
			//fmt.Printf("add word:%s, ts:%d\n", w,time.Now().UnixNano())
			//atomic.AddInt32(&f.produced, 1)
		default:
			log.Warn("too many pending words, len:%d lost:%s",
				len(f.wordChan), w/*, atomic.LoadInt32(&f.produced), atomic.LoadInt32(&f.consumed)*/)
			//os.Exit(-1)
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
		ts:           time.Now().Unix(),
		cb:           cb,
	}:
	default:
		log.Warn("too many pending get-freq tasks, lost:%d", lastNSeconds)
	}
}

func (f *Frequency) processTask(task *sortTask) {
	if task == nil {
		return
	}

	var (
		frqs      = make(wordmeta.Datas, 0, task.lastNSeconds)
		words     *wordsbysec.Words
		startNode = f.frqBySec.Back()
	)

	n2w := func(n *list.Element) *wordsbysec.Words {
		if n == nil || n.Value == nil {
			return nil
		}
		w, ok := n.Value.(*wordsbysec.Words)
		if !ok {
			return nil
		}
		return w
	}

	words = n2w(startNode)
	if words == nil {
		task.cb(nil, nil)
		return
	}
	offset := int(task.ts - words.TS)
	log.Release("offset:%d", offset)
	for i := 0; i < offset; i++ {
		if startNode == nil {
			break
		}
		startNode = startNode.Prev()
	}

	for i := 0; i < task.lastNSeconds; i++ {
		if startNode == nil {
			log.Release("i:%d, nil node", i)
			break
		}

		words = n2w(startNode)
		if words == nil {
			log.Release("i:%d, nil words", i)
			break
		}
		if words.MaxWord != nil {
			frqs = append(frqs, words.MaxWord)
		} else {
			log.Release("i:%d, nil max word, words:%d, ts:%d", i, words.GetWordCount(), words.TS)
		}

		startNode = startNode.Prev()
	}

	l := len(frqs)
	if l > 0 {
		if l > 1 {
			sort.Sort(frqs)
		}
		task.cb(frqs[0], nil)
	} else {
		task.cb(nil, nil)
	}
}
