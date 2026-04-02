package main

func mark(s []int, v int, ret bool) bool {
	s[0] += v
	return ret
}

func run() int {
	s := make([]int, 1)
	_ = mark(s, 1, false) && mark(s, 10, true)
	return s[0]
}
