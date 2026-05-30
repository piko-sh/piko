package main

type Counter struct {
	N int
}

func (c *Counter) Bump() {
	c.N++
}

func run() int {
	c := &Counter{N: 10}
	bump := (*Counter).Bump
	bump(c)
	bump(c)
	bump(c)
	return c.N
}
