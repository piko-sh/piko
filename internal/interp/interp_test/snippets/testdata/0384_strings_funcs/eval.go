package main

import "strings"

func run() string {
	return strings.ToUpper("hello") + "-" + strings.Repeat("x", 3)
}
