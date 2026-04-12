package main

import "time"

func run() int {
	t := time.NewTicker(2 * time.Millisecond)
	defer t.Stop()
	count := 0
	for i := 0; i < 3; i++ {
		<-t.C
		count++
	}
	return count
}
