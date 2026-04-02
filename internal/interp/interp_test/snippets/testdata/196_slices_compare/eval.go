package main

import "slices"

func run() int {
	a := []int{1, 2, 3}
	b := []int{1, 2, 4}
	c := []int{1, 2, 3}

	r := 0
	if slices.Compare(a, b) < 0 {
		r += 100
	}
	if slices.Compare(b, a) > 0 {
		r += 10
	}
	if slices.Compare(a, c) == 0 {
		r += 1
	}
	return r
}
