package main

func sendOne(ch chan<- int) {
	select {
	case ch <- 1:
	default:
	}
}

func run() int {
	ch := make(chan int, 1)
	sendOne(ch)
	return <-ch
}
