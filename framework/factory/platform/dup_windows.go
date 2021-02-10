package platform

import (
	"fmt"
	"syscall"
)

func DupExt(from int, to int) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setStdHandle := kernel32.NewProc("SetStdHandle")
	sh := syscall.STD_ERROR_HANDLE
	v, _, err := setStdHandle.Call(uintptr(sh), uintptr(from))
	if v == 0 {
		if err != nil {
			panic(fmt.Sprintf("DupExt failed %v", err.Error()))
		}
	}
}