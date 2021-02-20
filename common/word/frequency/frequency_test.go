package frequency

import "testing"

var fh = New()

func BenchmarkFrequency_Add(b *testing.B) {
	fh.Add("")
}
