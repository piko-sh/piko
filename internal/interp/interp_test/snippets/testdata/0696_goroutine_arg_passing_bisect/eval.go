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

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			counter.Add(int64(workerID + 1))
		}(i)
	}

	wg.Wait()

	expected := int64(0)
	for i := 0; i < workers; i++ {
		expected += int64(i + 1)
	}
	return fmt.Sprintf("count=%d/exp=%d", counter.Load(), expected)
}
