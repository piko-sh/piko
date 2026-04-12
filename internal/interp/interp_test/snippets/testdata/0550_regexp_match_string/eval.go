package main

import "regexp"

func run() int {
	re := regexp.MustCompile(`^abc`)
	if re.MatchString("abcdef") && !re.MatchString("xyz") {
		return 1
	}
	return 0
}
