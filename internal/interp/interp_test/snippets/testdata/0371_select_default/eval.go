package main

func run() string {
	ch := make(chan int, 1)
	ch <- 7
	select {
	case v := <-ch:
		_ = v
		return "received"
	default:
		return "none"
	}
}
