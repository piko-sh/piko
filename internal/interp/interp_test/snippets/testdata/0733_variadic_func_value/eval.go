package main

import "fmt"

func sumAll(nums ...int) int {
	s := 0
	for _, n := range nums {
		s += n
	}
	return s
}

func concatAll(parts ...string) string {
	out := ""
	for _, p := range parts {
		out += p
	}
	return out
}

func run() string {
	result := ""

	var f func(...int) int
	f = sumAll
	result += fmt.Sprintf("f3=%d;", f(1, 2, 3))
	result += fmt.Sprintf("f5=%d;", f(1, 2, 3, 4, 5))

	nums := []int{10, 20, 30}
	result += fmt.Sprintf("spread=%d;", f(nums...))

	var g func(...string) string
	g = concatAll
	result += fmt.Sprintf("g=%s", g("a", "b", "c"))

	return result
}
