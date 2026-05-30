package main

func run() (result int) {
	ch := make(chan int, 1)
	ch <- 7
	defer func() {
		select {
		case v := <-ch:
			result = v
		default:
			result = -1
		}
	}()
	return 0
}
