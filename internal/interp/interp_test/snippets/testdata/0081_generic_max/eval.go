package main

type Ordered interface {
	~int | ~float64 | ~string
}

func max2[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func run() int {
	return max2(3, 7)
}
