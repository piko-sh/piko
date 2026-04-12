package main

import (
	"errors"
	"sync"
)

func run() int {
	errs := make([]error, 3)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			if v != 1 {
				errs[v] = errors.New("oops")
			}
		}(i)
	}
	wg.Wait()
	joined := errors.Join(errs...)
	if joined != nil {
		return 1
	}
	return 0
}
