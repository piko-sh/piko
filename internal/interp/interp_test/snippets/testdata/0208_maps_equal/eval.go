package main

import "maps"

func run() int {
	a := map[string]int{"x": 1, "y": 2}
	b := map[string]int{"x": 1, "y": 2}
	c := map[string]int{"x": 1, "y": 3}

	r := 0
	if maps.Equal(a, b) {
		r += 10
	}
	if !maps.Equal(a, c) {
		r += 1
	}
	return r
}
