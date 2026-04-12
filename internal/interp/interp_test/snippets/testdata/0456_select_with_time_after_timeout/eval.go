package main

import "time"

func run() int {
	ch := make(chan int)
	select {
	case <-ch:
		return 0
	case <-time.After(20 * time.Millisecond):
		return 1
	}
}
