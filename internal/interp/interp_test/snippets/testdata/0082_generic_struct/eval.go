package main

type Pair[A any, B any] struct {
	First  A
	Second B
}

func run() int {
	p := Pair[int, int]{First: 10, Second: 20}
	return p.First + p.Second
}
