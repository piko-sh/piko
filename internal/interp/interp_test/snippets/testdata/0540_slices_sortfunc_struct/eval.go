package main

import (
	"cmp"
	"slices"
	"strconv"
)

type item struct {
	id    int
	value int
}

func run() string {
	s := []item{{3, 30}, {1, 10}, {2, 20}}
	slices.SortFunc(s, func(a, b item) int { return cmp.Compare(a.id, b.id) })
	out := ""
	for i, it := range s {
		if i > 0 {
			out += ","
		}
		out += strconv.Itoa(it.value)
	}
	return out
}
