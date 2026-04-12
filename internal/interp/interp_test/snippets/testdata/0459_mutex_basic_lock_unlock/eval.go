package main

import "sync"

func run() int {
	var mu sync.Mutex
	total := 0
	for i := 1; i <= 5; i++ {
		mu.Lock()
		total += i
		mu.Unlock()
	}
	return total
}
