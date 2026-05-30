package main

import "sync"

func run() int {
	var wg sync.WaitGroup
	wg.Wait()
	return 1
}
