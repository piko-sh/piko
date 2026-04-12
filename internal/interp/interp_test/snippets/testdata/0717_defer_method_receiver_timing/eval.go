package main

import "fmt"

type Counter struct {
	N int
}

func (c *Counter) PtrRead() int { return c.N }
func (c Counter) ValRead() int  { return c.N }

func deferPointerReceiver() (observed int) {
	c := &Counter{N: 1}
	defer func() {
		observed = c.PtrRead()
	}()
	c.N = 99
	return 0
}

func deferValueReceiver() (observed int) {
	c := Counter{N: 1}
	defer func() {
		observed = c.ValRead()
	}()
	c.N = 99
	return 0
}

func deferBoundValueMethodCallEvalEarly() (observed int) {
	c := Counter{N: 1}
	deferred := c.ValRead
	defer func() {
		observed = deferred()
	}()
	c.N = 99
	return 0
}

func deferBoundPointerMethodCallEvalEarly() (observed int) {
	c := &Counter{N: 1}
	deferred := c.PtrRead
	defer func() {
		observed = deferred()
	}()
	c.N = 99
	return 0
}

func run() string {
	a := deferPointerReceiver()
	b := deferValueReceiver()
	c := deferBoundValueMethodCallEvalEarly()
	d := deferBoundPointerMethodCallEvalEarly()
	return fmt.Sprintf("ptr=%d;val=%d;boundVal=%d;boundPtr=%d", a, b, c, d)
}
