package main

func sum() int {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)
	total := 0
	for v := range ch {
		total += v
	}
	return total
}

func run() int {
	return sum()
}
