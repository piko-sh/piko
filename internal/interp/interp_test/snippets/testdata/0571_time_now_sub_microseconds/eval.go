package main

import "time"

func run() int {
	start := time.Now()
	for i := 0; i < 100; i++ {
		_ = i
	}
	elapsed := time.Since(start)
	if elapsed >= 0 {
		return 1
	}
	return 0
}
