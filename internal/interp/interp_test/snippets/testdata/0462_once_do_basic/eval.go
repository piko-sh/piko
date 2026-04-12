package main

import "sync"

func run() int {
	var once sync.Once
	counter := 0
	f := func() { counter++ }
	once.Do(f)
	once.Do(f)
	once.Do(f)
	return counter
}
