package main

type counter struct {
	n int
}

func (c *counter) inc() {
	c.n++
}

func run() int {
	c := &counter{n: 5}
	c.inc()
	defer c.inc()
	return c.n
}
