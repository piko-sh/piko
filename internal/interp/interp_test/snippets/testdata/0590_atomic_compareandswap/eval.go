package main

import "sync/atomic"

func run() int {
	var n int64 = 0
	swapped := atomic.CompareAndSwapInt64(&n, 0, 42)
	if swapped && atomic.LoadInt64(&n) == 42 {
		return 1
	}
	return 0
}
