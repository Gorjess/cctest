package ostype

import "runtime"

const (
	Windows = "windows"
	Linux   = "linux"
	Darwin  = "darwin"
)

func Get() string {
	return runtime.GOOS
}
