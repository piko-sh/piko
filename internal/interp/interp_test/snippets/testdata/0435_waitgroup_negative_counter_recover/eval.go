package main

import "sync"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
			return
		}
		result = 99
	}()
	var wg sync.WaitGroup
	wg.Done()
	return
}
