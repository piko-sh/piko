package main

import "time"

func run() int {
	done := make(chan struct{})
	signal := make(chan int, 1)
	go func() {
		<-done
		signal <- 9
	}()
	result := 0
	select {
	case v := <-signal:
		result = v
	case <-time.After(30 * time.Millisecond):
		result = 1
	}
	close(done)
	<-signal
	return result
}
