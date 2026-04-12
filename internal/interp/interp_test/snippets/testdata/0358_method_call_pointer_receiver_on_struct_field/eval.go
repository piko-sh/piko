package main

type Counter struct {
	N int
}

func (c *Counter) Inc() {
	c.N++
}

type Holder struct {
	Inner Counter
}

func run() int {
	h := Holder{Inner: Counter{N: 10}}
	h.Inner.Inc()
	h.Inner.Inc()
	return h.Inner.N
}
