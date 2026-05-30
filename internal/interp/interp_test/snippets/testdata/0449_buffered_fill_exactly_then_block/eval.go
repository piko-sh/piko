package main

func run() int {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	select {
	case ch <- 4:
		return 0
	default:
		return 5
	}
}
