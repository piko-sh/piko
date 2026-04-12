package main

func run() int {
	ch := make(chan int, 1)
	ch <- 42
	close(ch)
	v1, ok1 := <-ch
	v2, ok2 := <-ch
	r := v1
	if !ok1 {
		r += 100
	}
	if ok2 {
		r += 200
	}
	_ = v2
	return r
}
