package main

import "time"

func run() int {
	ch := make(chan int, 1)
	t := time.AfterFunc(50*time.Millisecond, func() {
		ch <- 7
	})
	defer t.Stop()
	return <-ch
}
