package maxHeap

import (
	"strconv"
	"testing"
)

type CEntry int

func (ce CEntry) Value() interface{} {
	return ce
}

func (ce CEntry) String() string {
	return strconv.Itoa(int(ce))
}

var sli = func(l int) []IEntry {
	var (
		o = make([]IEntry, l)
	)
	for i := 0; i < l; i++ {
		o[i] = CEntry(i)
	}
	return o
}(1000)

func BenchmarkFromSlice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FromSlice(sli)
	}
}
