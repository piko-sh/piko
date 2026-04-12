package main

import "cmp"

func run() int {
	r := 0
	if cmp.Compare(1, 2) < 0 {
		r += 100
	}
	if cmp.Compare(5, 5) == 0 {
		r += 10
	}
	if cmp.Compare(9, 3) > 0 {
		r += 1
	}
	return r
}
