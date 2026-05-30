package main

import "sync"

type S struct{ x int }

func run() int {
	s := &S{x: 0}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			mu.Lock()
			s.x = s.x + v
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	return s.x
}
