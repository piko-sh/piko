package main

import "fmt"

func classify(x int) string {
	if x < -100 {
		return "very-negative"
	} else if x < 0 {
		return "negative"
	} else if x == 0 {
		return "zero"
	} else if x < 10 {
		return "small"
	} else if x < 100 {
		return "medium"
	} else if x < 1000 {
		return "large"
	} else {
		return "huge"
	}
}

func describe(v any) string {
	switch outer := v.(type) {
	case int:
		switch inner := any(outer * 2).(type) {
		case int:
			return fmt.Sprintf("int->%d", inner)
		case string:
			return fmt.Sprintf("int->str=%s", inner)
		default:
			return "int->unknown"
		}
	case string:
		switch outer {
		case "":
			return "empty"
		case "x":
			return "single-x"
		default:
			return "string-" + outer
		}
	default:
		return "unknown"
	}
}

func run() string {
	result := ""
	for _, v := range []int{-200, -5, 0, 7, 50, 500, 5000} {
		result += classify(v) + ";"
	}
	result += describe(7) + ";"
	result += describe("x") + ";"
	result += describe("hello") + ";"
	result += describe(3.14)
	return result
}
