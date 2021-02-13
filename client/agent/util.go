package agent

import (
	"fmt"
)

func printTitle() {
	fmt.Printf(`
*********************Client started*************************
`)
}

func pureLog(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Println()
}

func printChat(self, from, content string, ts string) {
	sender := from
	if self == from {
		sender = "You"
	}
	fmt.Printf("[%s] %s said: %s\n", ts, sender, content)
}
