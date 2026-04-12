package main

type Counter struct {
	N int
}

func (c *Counter) Add(v int) {
	c.N += v
}

func run() int {
	c := &Counter{N: 10}
	c.Add(5)
	c.Add(3)
	return c.N
}
