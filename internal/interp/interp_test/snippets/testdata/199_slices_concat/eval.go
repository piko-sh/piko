package main

import "slices"

func run() int {
	a := []int{1, 2}
	b := []int{3, 4}
	c := []int{5}
	s := slices.Concat(a, b, c)
	return len(s)*100000 + s[0]*10000 + s[1]*1000 + s[2]*100 + s[3]*10 + s[4]
}
