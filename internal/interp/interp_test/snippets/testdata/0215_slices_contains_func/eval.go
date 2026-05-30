package main

import "slices"

func run() bool {
	s := []int{1, 2, 3, 4, 5}
	return slices.ContainsFunc(s, func(v int) bool { return v > 3 })
}
