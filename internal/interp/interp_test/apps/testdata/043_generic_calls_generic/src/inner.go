package main

type Number interface {
	~int | ~float64
}

func tripled[T Number](v T) T {
	return v + v + v
}
