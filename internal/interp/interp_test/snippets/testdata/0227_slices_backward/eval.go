package main

import "slices"

func run() int {
	s := []int{1, 2, 3}
	result := 0
	for _, item := range slices.Backward(s) {
		result = result*10 + item
	}
	return result
}
