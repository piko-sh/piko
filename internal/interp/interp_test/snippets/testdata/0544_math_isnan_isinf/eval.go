package main

import "math"

func run() int {
	if math.IsNaN(math.NaN()) && math.IsInf(math.Inf(1), 1) && math.IsInf(math.Inf(-1), -1) {
		return 1
	}
	return 0
}
