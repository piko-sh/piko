package main

import "slices"

func run() int {
	s := []int{5, 3, 1, 4, 2}
	result := slices.Sorted(slices.Values(s))
	return result[0]*10000 + result[1]*1000 + result[2]*100 + result[3]*10 + result[4]
}
