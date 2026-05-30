package main

import (
	"fmt"
	"sort"
	"sync"
)

func run() string {
	var mu sync.Mutex
	cond := sync.NewCond(&mu)
	ready := false
	var results []int
	var resultsMu sync.Mutex

	var wg sync.WaitGroup
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mu.Lock()
			for !ready {
				cond.Wait()
			}
			mu.Unlock()
			resultsMu.Lock()
			results = append(results, id)
			resultsMu.Unlock()
		}(i)
	}

	mu.Lock()
	ready = true
	mu.Unlock()
	cond.Broadcast()

	wg.Wait()
	sort.Ints(results)
	return fmt.Sprintf("results=%v,len=%d", results, len(results))
}
