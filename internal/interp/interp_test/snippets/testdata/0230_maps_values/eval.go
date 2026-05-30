package main

import (
	"maps"
	"slices"
)

func run() int {
	m := map[string]int{"a": 10, "b": 20, "c": 30}
	var vals []int
	for v := range maps.Values(m) {
		vals = append(vals, v)
	}
	slices.Sort(vals)
	result := 0
	for _, v := range vals {
		result = result*100 + v
	}
	return result
}
