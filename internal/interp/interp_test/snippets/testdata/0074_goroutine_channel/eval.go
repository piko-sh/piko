package main

func run() int {
	ch := make(chan int, 1)
	go func() {
		ch <- 99
	}()
	v := <-ch
	return v
}
