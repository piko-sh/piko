package main

type Box struct {
	N int
}

func run() int {
	original := &Box{N: 7}
	alias := original
	alias.N = 99
	return original.N
}
