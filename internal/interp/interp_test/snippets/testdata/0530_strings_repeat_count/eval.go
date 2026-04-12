package main

import "strings"

func run() string {
	r := strings.Repeat("ab", 3)
	c := strings.Count("ababab", "ab")
	return r + ":" + string(rune('0'+c))
}
