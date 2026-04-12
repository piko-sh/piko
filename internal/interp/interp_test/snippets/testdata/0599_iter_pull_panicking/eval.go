package main

import "slices"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	s := []int{1, 2, 3}
	slices.DeleteFunc(s, func(v int) bool {
		panic("delete-func boom")
	})
	return 99
}
