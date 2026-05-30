package main

import "math"

func run() int {
	s := math.Sin(0)
	c := math.Cos(0)
	if s == 0 && c == 1 {
		return 1
	}
	return 0
}
