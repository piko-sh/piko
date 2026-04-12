package main

import (
	"fmt"
	"time"
)

func run() string {
	result := ""

	t := time.NewTimer(20 * time.Millisecond)
	<-t.C
	result += "fired1;"

	t.Reset(20 * time.Millisecond)
	<-t.C
	result += "fired2;"

	t2 := time.NewTimer(time.Hour)
	stopped := t2.Stop()
	result += fmt.Sprintf("stop_pending=%t;", stopped)

	t3 := time.NewTimer(5 * time.Millisecond)
	time.Sleep(30 * time.Millisecond)
	stoppedLate := t3.Stop()
	result += fmt.Sprintf("stop_after=%t", stoppedLate)

	return result
}
