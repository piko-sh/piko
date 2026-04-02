package main

import "strings"

func run() int {
	r := 0
	parts := strings.Split("a,b,c", ",")
	if len(parts) == 3 && parts[0] == "a" && parts[1] == "b" && parts[2] == "c" {
		r += 1
	}
	joined := strings.Join(parts, "-")
	if joined == "a-b-c" {
		r += 2
	}
	replaced := strings.ReplaceAll("hello world hello", "hello", "hi")
	if replaced == "hi world hi" {
		r += 4
	}
	return r
}
