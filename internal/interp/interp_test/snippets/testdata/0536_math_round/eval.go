package main

import "math"

func run() int {
	a := math.Round(2.5)
	b := math.Round(2.4)
	c := math.Round(-2.5)
	if a == 3 && b == 2 && c == -3 {
		return 1
	}
	return 0
}
