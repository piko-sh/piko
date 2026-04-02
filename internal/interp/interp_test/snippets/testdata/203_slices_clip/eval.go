package main

import "slices"

func run() int {
	s := make([]int, 3, 100)
	s[0] = 1
	s[1] = 2
	s[2] = 3
	s = slices.Clip(s)
	return len(s)*10 + s[0]
}
