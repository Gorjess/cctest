package agent

type Agent interface {
	OnClose(code uint)
	Addr() string
}
