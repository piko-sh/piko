package main

import "slices"

func run() int {
	original := []int{10, 20, 30}
	cloned := slices.Clone(original)
	cloned[0] = 99
	return original[0] + cloned[0]
}
