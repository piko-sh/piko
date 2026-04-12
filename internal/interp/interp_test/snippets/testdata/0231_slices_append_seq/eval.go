package main

import "slices"

func run() int {
	s := []int{1, 2, 3}
	iter := func(yield func(int) bool) {
		if !yield(4) {
			return
		}
		yield(5)
	}
	result := slices.AppendSeq(s, iter)
	sum := 0
	for _, v := range result {
		sum += v
	}
	return sum
}
