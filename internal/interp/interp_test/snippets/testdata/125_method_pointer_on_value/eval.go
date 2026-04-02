package main

type Counter struct {
	N int
}

func (c *Counter) Inc() {
	c.N++
}

func run() int {
	c := Counter{N: 10}
	c.Inc()
	c.Inc()
	return c.N
}
