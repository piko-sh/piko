package main

import "sort"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	s := []int{3, 1, 2}
	sort.Slice(s, func(i, j int) bool {
		panic("compare boom")
	})
	return 99
}
