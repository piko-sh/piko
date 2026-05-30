package main

func produce[T any](items []T) <-chan T {
	ch := make(chan T, len(items))
	for _, v := range items {
		ch <- v
	}
	close(ch)
	return ch
}

func run() int {
	ch := produce([]int{10, 20, 30})
	total := 0
	for v := range ch {
		total += v
	}
	return total
}
