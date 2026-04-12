package main

import "math"

func run() int {
	p := math.Inf(1)
	n := math.Inf(-1)
	if p > 1e308 && n < -1e308 && p != n {
		return 1
	}
	return 0
}
