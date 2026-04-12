package main

func identity[T any](v T) T {
	return v
}

func max2[T ~int](a, b T) T {
	if a > b {
		return a
	}
	return b
}
