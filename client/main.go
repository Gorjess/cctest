package main

import (
	"cloudcadetest/client/agent"
	_ "net/http/pprof"
)

func main() {
	//go func() {
	//	fmt.Println(http.ListenAndServe("localhost:6060", nil))
	//}()

	agent.New()
}
