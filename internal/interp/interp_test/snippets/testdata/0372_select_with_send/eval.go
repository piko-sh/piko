package main

func run() int {
	ch := make(chan int, 1)
	select {
	case ch <- 7:
		return 1
	default:
		return -1
	}
}
