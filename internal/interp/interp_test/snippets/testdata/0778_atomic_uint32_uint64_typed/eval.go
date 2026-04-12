package main

import (
	"fmt"
	"sync/atomic"
)

func run() string {
	var u32 atomic.Uint32
	u32.Store(100)
	u32.Add(50)
	swapped32 := u32.CompareAndSwap(150, 200)
	result := fmt.Sprintf("u32=%d,cas=%t;", u32.Load(), swapped32)

	var u64 atomic.Uint64
	u64.Store(1_000_000)
	u64.Add(500_000)
	prev := u64.Swap(42)
	result += fmt.Sprintf("u64_prev=%d,u64_now=%d", prev, u64.Load())
	return result
}
