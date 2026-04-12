package main

import "slices"

func run() int {
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}

	r := 0
	if slices.Equal(a, b) {
		r += 10
	}
	if !slices.Equal(a, c) {
		r += 1
	}
	return r
}
