package main

func run() int {
	outer := make(chan int, 1)
	go func() {
		inner := make(chan int, 1)
		go func() {
			inner <- 7
		}()
		outer <- <-inner
	}()
	return <-outer
}
