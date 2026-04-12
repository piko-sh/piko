package main

import "strings"

func run() string {
	parts := strings.Split("a,b,c,d", ",")
	return strings.Join(parts, "|")
}
