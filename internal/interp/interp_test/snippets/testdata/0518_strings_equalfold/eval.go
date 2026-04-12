package main

import "strings"

func run() int {
	if strings.EqualFold("Hello", "hELLO") {
		return 1
	}
	return 0
}
