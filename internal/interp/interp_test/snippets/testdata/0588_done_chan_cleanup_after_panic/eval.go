package main

import "sync"

func run() int {
	cleaned := 0
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				cleaned = 1
			}
		}()
		panic("oops")
	}()
	wg.Wait()
	return cleaned
}
