package main

func run() int {
	ch := make(chan int, 1)
	ch <- 7
	result := 0
	select {
	case v := <-ch:
		result = v
	default:
		result = -1
	}
	return result
}
