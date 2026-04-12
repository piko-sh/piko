package main

func mapAll[T any, U any](items []T, fn func(T) U) []U {
	out := make([]U, len(items))
	for i, v := range items {
		out[i] = fn(v)
	}
	return out
}

func run() int {
	doubled := mapAll([]int{1, 2, 3}, func(n int) int { return n * 2 })
	return doubled[0]
}
