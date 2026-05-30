package main

func run() int {
	ch := make(chan int)
	select {
	case ch <- 1:
		return 0
	default:
		return 4
	}
}
