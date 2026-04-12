package main

import "math"

func run() int {
	inf := math.Inf(1)
	if inf > 1e300 && inf == inf {
		return 1
	}
	return 0
}
