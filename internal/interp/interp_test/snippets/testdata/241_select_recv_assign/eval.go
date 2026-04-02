package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	v := 0
	select {
	case v = <-ch:
	default:
	}
	return v
}
