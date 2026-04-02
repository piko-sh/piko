package main

import "slices"

func run() int {
	a := []int{1, 2, 3}
	b := []int{1, 2, 4}
	return slices.CompareFunc(a, b, func(x, y int) int { return x - y })
}
