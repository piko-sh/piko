package main

type Box struct {
	N int
}

func run() int {
	ch := make(chan Box, 1)
	b := Box{N: 7}
	ch <- b
	b.N = 99
	r := <-ch
	return r.N
}
