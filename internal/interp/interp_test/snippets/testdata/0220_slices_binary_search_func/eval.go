package main

import (
	"cmp"
	"slices"
)

func run() int {
	s := []int{10, 20, 30, 40, 50}
	index, _ := slices.BinarySearchFunc(s, 30, func(a, b int) int {
		return cmp.Compare(a, b)
	})
	return index
}
