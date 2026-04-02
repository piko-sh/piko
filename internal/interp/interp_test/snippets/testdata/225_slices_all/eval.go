package main

import "slices"

func run() int {
	s := []int{10, 20, 30}
	sum := 0
	for i, v := range slices.All(s) {
		sum += i*100 + v
	}
	return sum
}
