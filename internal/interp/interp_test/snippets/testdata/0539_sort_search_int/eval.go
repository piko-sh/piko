package main

import "sort"

func run() int {
	s := []int{1, 3, 5, 7, 9, 11}
	i := sort.Search(len(s), func(i int) bool { return s[i] >= 7 })
	return i
}
