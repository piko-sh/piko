package main

import "slices"

func run() int {
	s := []int{10, 20, 30, 40, 50}
	index, found := slices.BinarySearch(s, 30)
	r := index * 10
	if found {
		r += 1
	}
	return r
}
