package main

type closer struct {
	closed bool
}

func (c *closer) Close() {
	c.closed = true
}

func run() int {
	c := &closer{}
	func() {
		defer c.Close()
	}()
	if c.closed {
		return 1
	}
	return 0
}
