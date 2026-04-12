package main

import "sync"

func run() int {
	const N = 1000
	ch := make(chan int, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			ch <- v
		}(i)
	}
	wg.Wait()
	close(ch)
	sum := 0
	for v := range ch {
		sum += v
	}
	return sum
}
