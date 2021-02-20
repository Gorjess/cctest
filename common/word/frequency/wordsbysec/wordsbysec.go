package wordsbysec

import "cloudcadetest/common/word/frequency/wordmeta"

type Words struct {
	maxWordNum int
	maxWord    *wordmeta.Data
	words      map[string]int
}

func New(maxWordNum int) *Words {
	return &Words{
		maxWordNum: maxWordNum,
		maxWord:    new(wordmeta.Data),
		words:      map[string]int{},
	}
}

func (w *Words) Add(word string) {
	// just refuse stat any more words
	if len(w.words) == w.maxWordNum {
		return
	}
	w.words[word]++

	cnt := w.words[word]
	if cnt > w.maxWord.Count {
		w.maxWord = &wordmeta.Data{
			Word:  word,
			Count: cnt,
		}
	}
}
