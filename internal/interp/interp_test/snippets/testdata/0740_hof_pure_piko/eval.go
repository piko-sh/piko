package main

import "fmt"

func mapInt(xs []int, f func(int) int) []int {
	out := make([]int, len(xs))
	for i, v := range xs {
		out[i] = f(v)
	}
	return out
}

func filterInt(xs []int, pred func(int) bool) []int {
	out := make([]int, 0, len(xs))
	for _, v := range xs {
		if pred(v) {
			out = append(out, v)
		}
	}
	return out
}

func reduce(xs []int, init int, f func(int, int) int) int {
	acc := init
	for _, v := range xs {
		acc = f(acc, v)
	}
	return acc
}

func run() string {
	xs := []int{1, 2, 3, 4, 5}

	doubled := mapInt(xs, func(n int) int { return n * 2 })
	result := fmt.Sprintf("doubled=%v;", doubled)

	evens := filterInt(xs, func(n int) bool { return n%2 == 0 })
	result += fmt.Sprintf("evens=%v;", evens)

	sum := reduce(xs, 0, func(a, b int) int { return a + b })
	result += fmt.Sprintf("sum=%d;", sum)

	chained := reduce(mapInt(xs, func(n int) int { return n * 2 }), 0, func(a, b int) int { return a + b })
	result += fmt.Sprintf("chain=%d", chained)

	return result
}
