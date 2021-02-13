package agent

import (
	"fmt"
	"time"
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

func printChat(self, from, content string, ts time.Time) {
	sender := from
	if self == from {
		sender = "You"
	}
	fmt.Printf("[%s] %s said: %s\n", ts.Format("2006-01-02 15:04:05"), sender, content)
}
