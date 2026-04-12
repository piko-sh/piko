package main

import (
	"context"
	"errors"
	"sync"
	"time"
)

func run() int {
	ctx, cancel := context.WithTimeoutCause(context.Background(), 200*time.Millisecond, errors.New("test"))
	defer cancel()

	jobs := make(chan int, 5)
	results := make(chan int, 5)
	var wg sync.WaitGroup
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case j, ok := <-jobs:
					if !ok {
						return
					}
					results <- j * 2
				}
			}
		}()
	}
	for i := 1; i <= 5; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	close(results)
	sum := 0
	for r := range results {
		sum += r
	}
	return sum
}
