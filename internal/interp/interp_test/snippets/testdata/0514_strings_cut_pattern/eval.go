package main

import "strings"

func run() string {
	before, after, found := strings.Cut("key=value", "=")
	if !found {
		return "no"
	}
	return before + "|" + after
}
