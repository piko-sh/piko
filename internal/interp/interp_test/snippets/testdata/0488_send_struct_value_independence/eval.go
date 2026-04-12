package main

type S struct{ x int }

func run() int {
	ch := make(chan S, 1)
	sender := S{x: 42}
	ch <- sender
	sender.x = 99
	received := <-ch
	return received.x
}
