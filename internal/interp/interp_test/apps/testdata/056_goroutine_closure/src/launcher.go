package main

import "sync"

func launchAdders(c *counter, count int, perAdd int) {
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.add(perAdd)
		}()
	}
	wg.Wait()
}
