package main

import "slices"

func run() int {
	iter := func(yield func(int) bool) {
		for i := range 5 {
			if !yield(i + 1) {
				return
			}
		}
	}
	result := slices.Collect(iter)
	sum := 0
	for _, v := range result {
		sum += v
	}
	return sum
}
