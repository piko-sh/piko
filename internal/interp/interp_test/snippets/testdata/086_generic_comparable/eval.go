package main

func contains[T comparable](s []T, v T) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func run() bool {
	return contains([]int{1, 2, 3}, 2)
}
