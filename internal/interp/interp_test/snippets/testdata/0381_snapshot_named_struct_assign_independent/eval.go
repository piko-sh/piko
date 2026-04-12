package main

type Wide struct {
	A, B, C, D int
}

func run() int {
	src := Wide{A: 1, B: 2, C: 3, D: 4}
	dst := src
	dst.A = 99
	return src.A*1000 + dst.A
}
