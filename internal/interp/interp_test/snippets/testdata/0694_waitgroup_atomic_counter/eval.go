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
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterationsPerWorker; j++ {
				counter.Add(int64(workerID + 1))
			}
		}(i)
	}

	wg.Wait()

	expected := int64(0)
	for i := 0; i < workers; i++ {
		expected += int64(i+1) * iterationsPerWorker
	}

	return fmt.Sprintf("count=%d/exp=%d/ok=%v", counter.Load(), expected, counter.Load() == expected)
}
