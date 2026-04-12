package main

import "math"

func run() int {
	m := math.Mod(10.5, 3)
	if m > 1.4 && m < 1.6 {
		return 1
	}
	return 0
}
