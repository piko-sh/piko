package main

type Box struct {
	N int
}

func mutate(b Box) {
	b.N = 999
}

func run() int {
	b := Box{N: 7}
	mutate(b)
	return b.N
}
