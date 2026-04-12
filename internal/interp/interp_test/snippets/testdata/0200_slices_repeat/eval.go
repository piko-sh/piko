package main

import "slices"

func run() int {
	s := slices.Repeat([]int{1, 2}, 3)
	return len(s)*1000000 + s[0]*100000 + s[1]*10000 + s[2]*1000 + s[3]*100 + s[4]*10 + s[5]
}
