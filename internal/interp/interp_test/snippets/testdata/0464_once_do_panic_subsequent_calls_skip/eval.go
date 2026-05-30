package main

import "sync"

var counter int

func tryFirst(once *sync.Once) {
	defer func() { recover() }()
	once.Do(func() {
		counter = counter + 1
		panic("first")
	})
}

func run() int {
	counter = 0
	var once sync.Once
	tryFirst(&once)
	once.Do(func() {
		counter = counter + 1
	})
	return counter
}
