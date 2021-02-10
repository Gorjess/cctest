package network

type IProcessor interface {
	// must goroutine safe
	Marshal(msg interface{}) ([]byte, error)

	Unmarshal(data []byte, msg interface{}) error

	Route(id interface{}, args ...interface{}) error
}
