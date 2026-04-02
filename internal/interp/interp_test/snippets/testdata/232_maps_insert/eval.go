package main

import (
	"maps"
	"slices"
	"strings"
)

func run() string {
	m1 := map[string]int{"a": 1}
	iter := func(yield func(string, int) bool) {
		if !yield("b", 2) {
			return
		}
		yield("c", 3)
	}
	maps.Insert(m1, iter)
	var keys []string
	for k := range maps.Keys(m1) {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return strings.Join(keys, ",")
}
