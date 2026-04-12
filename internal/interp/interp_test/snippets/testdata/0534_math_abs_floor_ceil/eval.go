package main

import "math"

func run() int {
	a := math.Abs(-3.7)
	b := math.Floor(3.7)
	c := math.Ceil(3.2)
	if a == 3.7 && b == 3 && c == 4 {
		return 1
	}
	return 0
}
