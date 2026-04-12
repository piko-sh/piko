package main

func run() int {
	outer := make(chan chan int, 1)
	go func() {
		inner := make(chan int, 1)
		inner <- 13
		outer <- inner
	}()
	innerCh := <-outer
	return <-innerCh
}
