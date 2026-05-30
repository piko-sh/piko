package main

type Counter struct {
	value int
}

func (c Counter) Read() int {
	return c.value
}

func (c *Counter) Bump() {
	c.value++
}
