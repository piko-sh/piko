package main

import "slices"

func run() string {
	s := []string{"cherry", "apple", "banana"}
	return slices.MinFunc(s, func(a, b string) int {
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
}
