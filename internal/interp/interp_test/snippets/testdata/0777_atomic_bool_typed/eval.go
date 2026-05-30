package main

import (
	"fmt"
	"sync/atomic"
)

func run() string {
	var b atomic.Bool
	result := ""
	result += fmt.Sprintf("init=%t;", b.Load())

	b.Store(true)
	result += fmt.Sprintf("after_store=%t;", b.Load())

	swapped := b.CompareAndSwap(true, false)
	result += fmt.Sprintf("cas_true_false=%t,now=%t;", swapped, b.Load())

	failed := b.CompareAndSwap(true, false)
	result += fmt.Sprintf("cas_fail=%t,now=%t;", failed, b.Load())

	prev := b.Swap(true)
	result += fmt.Sprintf("swap=%t,now=%t", prev, b.Load())
	return result
}
