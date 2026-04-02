package main

type Number interface {
	~int | ~float64
}

func sum[T Number](a, b, c T) T {
	return a + b + c
}

func run() int {
	return sum(10, 20, 12)
}
