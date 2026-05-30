package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func run() string {
	var counter atomic.Int64
	var wg sync.WaitGroup

	const workers = 8
	const iterationsPerWorker = 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterationsPerWorker; j++ {
				counter.Add(1)
			}
		}()
	}

	wg.Wait()

	expected := int64(workers * iterationsPerWorker)
	return fmt.Sprintf("count=%d/exp=%d", counter.Load(), expected)
}
