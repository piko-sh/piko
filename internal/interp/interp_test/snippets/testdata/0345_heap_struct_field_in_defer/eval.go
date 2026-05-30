package main

type Counter struct {
	Value int
	Seen  int
}

func observe() Counter {
	c := Counter{Value: 5, Seen: 0}
	defer func() {
		c.Value = c.Value * 2
		c.Seen = 99
	}()
	c.Value = 10
	return c
}

func run() int {
	c := observe()
	return c.Value*100 + c.Seen
}
