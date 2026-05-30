package main

import "sync"

func run() int {
	x := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			x = x + 1
			mu.Unlock()
		}()
	}
	wg.Wait()
	return x
}
