package main

import "sync"

func run() int {
	var m sync.Map
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(key int) {
			defer wg.Done()
			m.Store(key, key*10)
		}(i)
	}
	wg.Wait()
	total := 0
	for i := 0; i < 4; i++ {
		v, ok := m.Load(i)
		if !ok {
			return -1
		}
		total += v.(int)
	}
	return total
}
