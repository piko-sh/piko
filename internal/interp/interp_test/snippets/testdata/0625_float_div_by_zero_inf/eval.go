package main

import "math"

func run() int {
	one := 1.0
	zero := 0.0
	r := one / zero
	if math.IsInf(r, 1) {
		return 1
	}
	return 0
}
