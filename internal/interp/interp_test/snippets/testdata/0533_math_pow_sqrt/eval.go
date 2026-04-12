package main

import "math"

func run() int {
	a := math.Pow(2, 10)
	b := math.Sqrt(144)
	if a == 1024 && b == 12 {
		return 1
	}
	return 0
}
