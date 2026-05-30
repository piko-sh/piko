package main

import "fmt"

func run() string {
	result := ""
	x := 5
	if x > 0 {
		goto positive
	}
	result += "negative-path;"
positive:
	result += "positive-path;"

	for i := 0; i < 3; i++ {
		if i == 1 {
			goto skip
		}
		result += fmt.Sprintf("iter=%d;", i)
	skip:
	}

	return result
}
