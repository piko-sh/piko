package main

import "slices"

func run() int {
	s := []int{1, 2, 3}
	s = slices.Grow(s, 100)
	return len(s)*10 + s[2]
}
