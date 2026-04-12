package main

func producer(out chan<- int, n int) {
	for i := 1; i <= n; i++ {
		out <- i
	}
	close(out)
}

func consumer(in <-chan int) int {
	sum := 0
	for v := range in {
		sum += v
	}
	return sum
}

func run() int {
	ch := make(chan int, 5)
	go producer(ch, 5)
	return consumer(ch)
}
