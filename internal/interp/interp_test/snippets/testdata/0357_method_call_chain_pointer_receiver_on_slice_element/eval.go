package main

type Counter struct {
	N int
}

func (c *Counter) Inc() {
	c.N++
}

func run() int {
	s := []Counter{{N: 0}, {N: 0}}
	s[0].Inc()
	s[0].Inc()
	s[1].Inc()
	return s[0].N*10 + s[1].N
}
