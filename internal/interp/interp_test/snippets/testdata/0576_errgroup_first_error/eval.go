package main

import (
	"errors"
	"sync"
)

func run() int {
	errs := make(chan error, 3)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(v int) {
			defer wg.Done()
			if v == 1 {
				errs <- errors.New("task-1 failed")
				return
			}
			errs <- nil
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			return 1
		}
	}
	return 0
}
