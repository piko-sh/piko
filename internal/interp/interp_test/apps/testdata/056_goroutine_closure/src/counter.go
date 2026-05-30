package main

import "sync"

type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) add(v int) {
	c.mu.Lock()
	c.n += v
	c.mu.Unlock()
}

func (c *counter) value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
