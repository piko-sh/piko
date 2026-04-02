package main

import (
	"maps"
	"slices"
	"strings"
)

func run() string {
	m := map[string]int{"cherry": 3, "apple": 1, "banana": 2}
	var keys []string
	for k := range maps.Keys(m) {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return strings.Join(keys, ",")
}
