package main

import "sync"

func run() int {
	var once sync.Once
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			once.Do(func() {
				mu.Lock()
				counter++
				mu.Unlock()
			})
		}()
	}
	wg.Wait()
	return counter
}
