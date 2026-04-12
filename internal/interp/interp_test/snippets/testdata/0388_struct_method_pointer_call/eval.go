package main

type Cell struct{ N int }

func (c *Cell) DoubleAndAdd(x int) int {
	c.N *= 2
	c.N += x
	return c.N
}

func run() int {
	c := Cell{N: 10}
	c.DoubleAndAdd(5)
	c.DoubleAndAdd(2)
	return c.N
}
