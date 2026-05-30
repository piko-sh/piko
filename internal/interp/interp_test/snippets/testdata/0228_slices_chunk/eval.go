package main

import "slices"

func run() int {
	s := []int{1, 2, 3, 4, 5}
	count := 0
	total := 0
	for chunk := range slices.Chunk(s, 2) {
		count++
		for _, v := range chunk {
			total += v
		}
	}
	return count*100 + total
}
