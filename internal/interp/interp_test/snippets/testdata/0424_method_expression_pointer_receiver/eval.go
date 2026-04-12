package main

type Counter struct {
	value int
}

func (c *Counter) Bump() {
	c.value++
}

func apply(bump func(*Counter), c *Counter) {
	bump(c)
}

func run() int {
	c := Counter{value: 5}
	bumpExpression := (*Counter).Bump
	apply(bumpExpression, &c)
	return c.value
}
