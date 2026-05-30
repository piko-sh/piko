package main

import "sync"

func run() int {
	const max = 3
	sem := make(chan struct{}, max)
	completed := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			mu.Lock()
			completed++
			mu.Unlock()
		}()
	}
	wg.Wait()
	return completed
}
