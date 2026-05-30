package main

import "sync"

func run() int {
	done := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex
	awake := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-done
			mu.Lock()
			awake++
			mu.Unlock()
		}()
	}
	close(done)
	wg.Wait()
	return awake
}
