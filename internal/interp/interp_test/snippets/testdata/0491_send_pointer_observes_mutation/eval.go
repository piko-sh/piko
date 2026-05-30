package main

type S struct{ x int }

func run() int {
	ch := make(chan *S, 1)
	s := &S{x: 1}
	ch <- s
	s.x = 99
	received := <-ch
	return received.x
}
