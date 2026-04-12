package main

import "sync"

func run() int {
	counter := 0
	init := sync.OnceFunc(func() {
		counter++
	})
	init()
	init()
	init()
	return counter
}
