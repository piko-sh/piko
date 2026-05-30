package main

type Counter struct {
	N int
}

func (c *Counter) Inc() {
	c.N++
}

func run() int {
	c := Counter{N: 0}
	f := func() int { return c.N }
	c.Inc()
	c.Inc()
	c.Inc()
	return f()
}
