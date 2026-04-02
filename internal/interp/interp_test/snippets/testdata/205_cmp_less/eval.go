package main

import "cmp"

func run() int {
	r := 0
	if cmp.Less("alpha", "beta") {
		r += 10
	}
	if !cmp.Less("beta", "alpha") {
		r += 1
	}
	return r
}
