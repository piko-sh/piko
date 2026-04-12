package main

import "sync"

func run() int {
	s := []int{1, 2}
	sp := &s
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		*sp = append(*sp, 3, 4, 5)
	}()
	wg.Wait()
	return len(*sp)
}
