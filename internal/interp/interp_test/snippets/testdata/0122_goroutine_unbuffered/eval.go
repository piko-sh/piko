package main

func run() int {
	ch := make(chan int)
	go func() {
		ch <- 42
	}()
	return <-ch
}
