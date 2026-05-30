package main

import "fmt"

func three() (int, string, bool) {
	return 7, "seven", true
}

func twoStrings() (string, string) {
	return "alpha", "beta"
}

func formatAll(args ...any) string {
	out := ""
	for _, a := range args {
		out += fmt.Sprintf("%v,", a)
	}
	return out
}

func sumAll(values ...int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}

func run() string {
	result := ""

	result += "any3:" + formatAll(three())

	result += ";formatTwo:" + formatAll(twoStrings())

	prefix := "head"
	_ = prefix

	result += ";three_in_print:" + fmt.Sprintln(three())

	return result
}
