package main

import "slices"

func run() int {
	s := []int{5, 3, 1, 4, 2}
	slices.SortStableFunc(s, func(a, b int) int { return a - b })
	return s[0]*10000 + s[1]*1000 + s[2]*100 + s[3]*10 + s[4]
}
