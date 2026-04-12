package main

import "sync"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 2
			return
		}
		result = 99
	}()
	var wg sync.WaitGroup
	wg.Add(-1)
	return
}
