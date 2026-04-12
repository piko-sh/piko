package main

import "slices"

func run() int {
	s := []int{1, 5}
	s = slices.Insert(s, 1, 2, 3, 4)
	return len(s)*100000 + s[0]*10000 + s[1]*1000 + s[2]*100 + s[3]*10 + s[4]
}
