package main

func run() int {
	var ch chan int
	select {
	case <-ch:
		return 0
	case ch <- 1:
		return 0
	default:
		return 1
	}
}
