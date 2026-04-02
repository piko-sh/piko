package main

func run() int {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)
	fns := make([]func() int, 3)
	idx := 0
	for v := range ch {
		v := v
		i := idx
		fns[i] = func() int { return v }
		idx++
	}
	return fns[0]() + fns[1]() + fns[2]()
}
