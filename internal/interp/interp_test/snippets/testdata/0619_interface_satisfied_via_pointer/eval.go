package main

type Counter interface {
	Inc()
	Get() int
}

type box struct {
	n int
}

func (b *box) Inc()     { b.n++ }
func (b *box) Get() int { return b.n }

func run() int {
	var c Counter = &box{}
	c.Inc()
	c.Inc()
	c.Inc()
	return c.Get()
}
