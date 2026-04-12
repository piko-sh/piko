package main

import "strings"

func run() string {
	return strings.ReplaceAll("foo bar foo baz foo", "foo", "X")
}
