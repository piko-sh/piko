package main

import "slices"

func run() int {
	s := []int{10, 20, 30}
	sum := 0
	for v := range slices.Values(s) {
		sum += v
	}
	return sum
}
