package main

import "regexp"

func run() string {
	re := regexp.MustCompile(`\d+`)
	return re.ReplaceAllString("foo 42 bar 99", "N")
}
