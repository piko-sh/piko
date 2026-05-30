package main

import (
	"sort"
	"strconv"
)

type pair struct {
	key int
	idx int
}

func run() string {
	s := []pair{{1, 0}, {2, 1}, {1, 2}, {2, 3}}
	sort.SliceStable(s, func(i, j int) bool { return s[i].key < s[j].key })
	out := ""
	for i, p := range s {
		if i > 0 {
			out += ","
		}
		out += strconv.Itoa(p.idx)
	}
	return out
}
