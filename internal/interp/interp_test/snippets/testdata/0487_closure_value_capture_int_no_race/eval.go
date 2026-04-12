package main

import "sync"

func run() int {
	ch := make(chan int, 100)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
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
