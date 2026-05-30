package main

import (
	"sync"
	"sync/atomic"
)

func run() int {
	var n int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&n, 1)
		}()
	}
	wg.Wait()
	return int(atomic.LoadInt64(&n))
}
