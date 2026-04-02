package main

import "slices"

func run() bool {
	s := []int{5, 4, 3, 2, 1}
	return slices.IsSortedFunc(s, func(a, b int) int { return b - a })
}
