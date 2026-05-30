package main

import "slices"

func run() int {
	s := []int{1, 1, 2, 2, 3, 3}
	s = slices.CompactFunc(s, func(a, b int) bool { return a == b })
	result := 0
	for _, v := range s {
		result = result*10 + v
	}
	return result
}
