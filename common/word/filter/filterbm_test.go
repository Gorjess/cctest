package filter

import (
	"bytes"
	"cloudcadetest/framework/module"
	"testing"
)

type CFilterSkeleton int

func newCFS() *CFilterSkeleton {
	return new(CFilterSkeleton)
}

func (cfs *CFilterSkeleton) GetServerModule() *module.ServerMod {
	return nil
}

func (cfs *CFilterSkeleton) GetID() int64 {
	return 0
}

func (cfs *CFilterSkeleton) GetWordListFilePath() string {
	return "list.txt"
}

func makeInputString() string {
	var b bytes.Buffer
	for i := 0; i < 100; i++ {
		b.WriteString("h")
	}
	return b.String()
}

func BenchmarkFilter_Check(b *testing.B) {
	f := New(newCFS())
	for i := 0; i < b.N; i++ {
		f.Check(makeInputString(), func(newStr string) {
			// foo-bar
		})
	}
}
