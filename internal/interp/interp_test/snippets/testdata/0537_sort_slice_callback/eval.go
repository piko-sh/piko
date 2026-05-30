package main

import (
	"sort"
	"strconv"
)

func run() string {
	s := []int{3, 1, 4, 1, 5, 9, 2, 6}
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += strconv.Itoa(v)
	}
	return out
}
