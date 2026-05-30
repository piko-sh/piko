package main

import "sync"

func run() int {
	merged := make(chan int, 9)
	var wg sync.WaitGroup
	for p := 1; p <= 3; p++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < 3; i++ {
				merged <- base*10 + i
			}
		}(p)
	}
	go func() {
		wg.Wait()
		close(merged)
	}()
	sum := 0
	for v := range merged {
		sum += v
	}
	return sum
}
