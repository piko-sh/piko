package main

func run() int {
	ch1 := make(chan int, 1)
	ch2 := make(chan int, 1)
	ch2 <- 42
	r := 0
	select {
	case v := <-ch1:
		r = v
	case v := <-ch2:
		r = v
	}
	return r
}
