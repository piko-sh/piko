package main

import "sync"

func run() int {
	var rw sync.RWMutex
	var wg sync.WaitGroup
	shared := 0
	writes := 0
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			rw.Lock()
			shared = v
			writes++
			rw.Unlock()
		}(i + 1)
	}
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rw.RLock()
			_ = shared
			rw.RUnlock()
		}()
	}
	wg.Wait()
	if writes == 5 && shared >= 1 && shared <= 5 {
		return 1
	}
	return 0
}
