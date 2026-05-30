package main

type ordered interface {
	~int | ~float64 | ~string
}

func maxOf[T ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
