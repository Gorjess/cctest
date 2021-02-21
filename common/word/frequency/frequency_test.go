package frequency

import (
	"cloudcadetest/common/word/frequency/wordmeta"
	"fmt"
	"testing"
	"time"
)

var fh = New()

func insertWords(n int) {
	for i := 0; i < n; i++ {
		fh.Add("hello")
	}
}

func BenchmarkFrequency_GetFrequencyByTime(b *testing.B) {
	for j := 0; j < b.N; j++ {
		insertWords(1000)

		exit := false
		t1 := time.Now()
		fh.GetFrequencyByTime(5, func(meta *wordmeta.Data, e error) {
			if meta != nil {
				fmt.Println(meta.Word, meta.Count, e)
			} else {
				fmt.Println("no word")
			}
			exit = true
		})

		for !exit {
			if exit {
				fmt.Println("dur:", time.Now().Sub(t1).Nanoseconds())
				break
			}
		}
	}
}

func BenchmarkFrequency_Add(b *testing.B) {
	for i := 1; i < b.N; i++ {
		insertWords(1000)
	}
}
