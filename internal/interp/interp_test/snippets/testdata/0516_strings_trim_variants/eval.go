package main

import "strings"

func run() string {
	a := strings.TrimSpace("  hello  ")
	b := strings.TrimPrefix("foo-bar", "foo-")
	c := strings.TrimSuffix("foo-bar", "-bar")
	d := strings.Trim("##hi##", "#")
	return a + "|" + b + "|" + c + "|" + d
}
