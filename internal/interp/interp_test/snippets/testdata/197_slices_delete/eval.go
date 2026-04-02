package main

import "slices"

func run() int {
	s := []int{10, 20, 30, 40, 50}
	s = slices.Delete(s, 1, 3)
	return len(s)*1000 + s[0]*100 + s[1]*10 + s[2]
}
