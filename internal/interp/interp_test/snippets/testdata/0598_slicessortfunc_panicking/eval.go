package main

import "slices"

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	s := []string{"c", "a", "b"}
	slices.SortFunc(s, func(a, b string) int {
		panic("cmp boom")
	})
	return 99
}
