package maxHeap

import (
	"fmt"
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

func makeSlice() []IEntry {
	var o = make([]IEntry, 100)
	for i := 0; i < 100; i++ {
		o[i] = CEntry(i)
	}

	return o
}

func BenchmarkFromSlice(b *testing.B) {
	sli := makeSlice()
	for i := 0; i < 1; i++ {
		n := FromSlice(sli)
		fmt.Println(Literal(n))
	}
}
