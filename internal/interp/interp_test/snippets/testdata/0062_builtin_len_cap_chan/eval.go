package main

func run() int {
	ch := make(chan int, 5)
	ch <- 1
	ch <- 2
	ch <- 3
	return len(ch)*10 + cap(ch)
}
