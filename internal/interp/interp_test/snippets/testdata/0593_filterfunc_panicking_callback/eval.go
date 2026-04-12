package main

import "slices"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	s := []int{1, 2, 3, 4}
	_ = slices.IndexFunc(s, func(v int) bool {
		panic("predicate boom")
	})
	return 99
}
