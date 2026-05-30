package main

func run() int {
	var ch chan int
	select {
	case <-ch:
		return 0
	default:
		return 3
	}
}
