package main

import "math"

func run() int {
	n := math.NaN()
	if n == n {
		return 0
	}
	return 1
}
