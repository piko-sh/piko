package main

import (
	"cmp"
	"slices"
)

type p struct{ n int }

func run() int {
	s := []p{{3}, {1}, {4}, {1}, {5}}
	minp := slices.MinFunc(s, func(a, b p) int { return cmp.Compare(a.n, b.n) })
	maxp := slices.MaxFunc(s, func(a, b p) int { return cmp.Compare(a.n, b.n) })
	return minp.n*100 + maxp.n
}
