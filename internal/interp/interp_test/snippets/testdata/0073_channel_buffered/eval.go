package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	v := <-ch
	return v
}
