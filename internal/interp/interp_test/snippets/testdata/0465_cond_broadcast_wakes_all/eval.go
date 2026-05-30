package main

import "sync"

func run() int {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	parkedCount := 0
	woken := 0
	parkedCh := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			parkedCount++
			if parkedCount == 3 {
				close(parkedCh)
			}
			cond.Wait()
			woken++
			mu.Unlock()
		}()
	}
	<-parkedCh
	mu.Lock()
	cond.Broadcast()
	mu.Unlock()
	wg.Wait()
	return woken
}
