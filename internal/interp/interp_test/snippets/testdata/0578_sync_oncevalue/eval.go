package main

import "sync"

func run() int {
	calls := 0
	provider := sync.OnceValue(func() int {
		calls++
		return 42
	})
	a := provider()
	b := provider()
	c := provider()
	if a == 42 && b == 42 && c == 42 && calls == 1 {
		return 1
	}
	return 0
}
