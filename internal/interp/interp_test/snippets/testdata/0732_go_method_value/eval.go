package main

import (
	"fmt"
	"sync"
)

type Counter struct {
	mu    sync.Mutex
	count int
}

func (c *Counter) Inc(by int) {
	c.mu.Lock()
	c.count += by
	c.mu.Unlock()
}

func run() string {
	result := ""

	c := &Counter{}
	var wg sync.WaitGroup
	method := c.Inc
	for i := 1; i <= 5; i++ {
		wg.Add(1)
		go func(delta int) {
			defer wg.Done()
			method(delta)
		}(i)
	}
	wg.Wait()
	result += fmt.Sprintf("methodvalue=%d", c.count)

	return result
}
