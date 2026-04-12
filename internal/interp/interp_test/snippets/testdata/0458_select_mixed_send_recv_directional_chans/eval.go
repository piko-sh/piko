package main

func run() int {
	ch := make(chan int, 1)
	var sendCh chan<- int = ch
	var recvCh <-chan int = ch
	_ = sendCh
	select {
	case sendCh <- 5:
	default:
	}
	select {
	case v := <-recvCh:
		return v
	default:
		return -1
	}
}
