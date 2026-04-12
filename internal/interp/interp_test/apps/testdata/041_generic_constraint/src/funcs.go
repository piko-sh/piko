package main

func maxOf[T Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}
