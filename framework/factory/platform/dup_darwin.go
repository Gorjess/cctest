package platform

import "syscall"

func DupExt(from int, to int) {
	syscall.Dup2(from, to)
}
