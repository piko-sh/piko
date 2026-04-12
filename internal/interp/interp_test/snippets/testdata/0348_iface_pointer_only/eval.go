package main

type Counter interface {
	Inc()
	Get() int
}

type cell struct{ N int }

func (c *cell) Inc()     { c.N++ }
func (c *cell) Get() int { return c.N }

func run() int {
	var c Counter = &cell{}
	c.Inc()
	c.Inc()
	c.Inc()
	return c.Get()
}
