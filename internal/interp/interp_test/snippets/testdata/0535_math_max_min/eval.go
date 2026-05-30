package main

import "math"

func run() int {
	a := math.Max(3, 5)
	b := math.Min(3, 5)
	if a == 5 && b == 3 {
		return 1
	}
	return 0
}
