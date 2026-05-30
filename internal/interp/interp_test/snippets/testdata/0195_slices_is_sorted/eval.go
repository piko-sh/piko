package main

import "slices"

func run() int {
	r := 0
	if slices.IsSorted([]int{1, 2, 3, 4, 5}) {
		r += 10
	}
	if !slices.IsSorted([]int{1, 3, 2}) {
		r += 1
	}
	return r
}
