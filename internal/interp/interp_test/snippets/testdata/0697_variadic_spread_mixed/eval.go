package main

import "fmt"

func sum(values ...int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}

func formatAll(prefix string, values ...any) string {
	result := prefix + ":"
	for _, v := range values {
		result += fmt.Sprintf("%v,", v)
	}
	return result
}

func passThrough(values ...int) int {
	return sum(values...)
}

func run() string {
	result := ""

	result += fmt.Sprintf("zero:%d;", sum())
	result += fmt.Sprintf("positional:%d;", sum(1, 2, 3, 4))
	result += fmt.Sprintf("spread:%d;", sum([]int{5, 6, 7}...))
	result += fmt.Sprintf("emptySpread:%d;", sum([]int{}...))

	var nilSlice []int
	result += fmt.Sprintf("nilSpread:%d;", sum(nilSlice...))

	result += fmt.Sprintf("passThrough:%d;", passThrough(10, 20, 30))

	result += "fmtAll:" + formatAll("nums", 1, "two", 3.0, true)

	mixed := []any{"x", 42, false}
	result += ";fmtSpread:" + formatAll("vars", mixed...)

	return result
}
