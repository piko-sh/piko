package main

import "slices"

func run() int {
	s := []int{1, 1, 2, 2, 2, 3, 3, 4}
	s = slices.Compact(s)
	return len(s)*100 + s[0]*10 + s[3]
}
