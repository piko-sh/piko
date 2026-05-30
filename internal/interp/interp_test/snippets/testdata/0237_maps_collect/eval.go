package main

import (
	"maps"
	"slices"
	"strings"
)

func run() string {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	m2 := maps.Collect(maps.All(m))
	var keys []string
	for k := range m2 {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return strings.Join(keys, ",")
}
