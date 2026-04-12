package main

import "time"

func run() int {
	done := make(chan int, 1)
	go func() {
		time.Sleep(5 * time.Millisecond)
		done <- 1
	}()
	return <-done
}
