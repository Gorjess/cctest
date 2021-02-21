package wordsbysec

import (
	"cloudcadetest/common/word/frequency/wordmeta"
	"time"
)

type Words struct {
	maxWordNum int
	MaxWord    *wordmeta.Data
	words      map[string]int
	TS         int64
}

func New(maxWordNum int) *Words {
	return &Words{
		maxWordNum: maxWordNum,
		words:      map[string]int{},
		TS:         time.Now().Unix(),
	}
}

func (w *Words) GetWordCount() int {
	return len(w.words)
}

func (w *Words) Add(word string) {
	// just refuse stat any more words
	if len(w.words) == w.maxWordNum {
		return
	}
	w.words[word]++

	cnt := w.words[word]
	if w.MaxWord == nil || cnt > w.MaxWord.Count {
		w.MaxWord = wordmeta.New(word, cnt)
	}
}
