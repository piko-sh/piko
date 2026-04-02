package main

import "slices"

func run() int {
	s := []int{1, 2, 3, 4, 5}
	s = slices.DeleteFunc(s, func(v int) bool { return v%2 == 0 })
	result := 0
	for _, v := range s {
		result = result*10 + v
	}
	return result
}
