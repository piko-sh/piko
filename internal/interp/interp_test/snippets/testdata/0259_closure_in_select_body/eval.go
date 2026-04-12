package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	var f func() int
	select {
	case v := <-ch:
		f = func() int { return v }
	}
	return f()
}
