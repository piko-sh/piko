package main

func filter[T any](xs []T, keep func(T) bool) []T {
	var out []T
	for _, v := range xs {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

func run() int {
	xs := filter([]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, func(n int) bool { return n%2 == 0 })
	total := 0
	for _, v := range xs {
		total += v
	}
	return total
}
