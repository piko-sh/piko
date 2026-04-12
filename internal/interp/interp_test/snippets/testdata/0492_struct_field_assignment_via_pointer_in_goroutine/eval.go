package main

import "sync"

type S struct{ x int }

func run() int {
	s := &S{}
	var wg sync.WaitGroup
	wg.Add(1)
	go func(target *S) {
		defer wg.Done()
		target.x = 77
	}(s)
	wg.Wait()
	return s.x
}
