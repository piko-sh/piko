package main

import "regexp"

func run() string {
	re := regexp.MustCompile(`\d+`)
	return re.FindString("hello 42 world 99")
}
