package main

func run() int {
	gen := make(chan int)
	squared := make(chan int)
	go func() {
		for i := 1; i <= 5; i++ {
			gen <- i
		}
		close(gen)
	}()
	go func() {
		for v := range gen {
			squared <- v * v
		}
		close(squared)
	}()
	sum := 0
	for v := range squared {
		sum += v
	}
	return sum
}
