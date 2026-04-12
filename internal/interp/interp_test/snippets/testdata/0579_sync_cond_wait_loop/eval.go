package main

import "sync"

func run() int {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	ready := false
	woke := 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		mu.Lock()
		for !ready {
			cond.Wait()
		}
		woke = 1
		mu.Unlock()
	}()
	mu.Lock()
	ready = true
	cond.Signal()
	mu.Unlock()
	wg.Wait()
	return woke
}
