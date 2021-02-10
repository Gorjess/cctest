package aes

import (
	"testing"
)

func TestMakeKey16(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Log(MakeKey16())
	}

	t.Log(len(DefaultKey))
	t.Log(DefaultKey)
}
