package main

import (
	"regexp"
	"strings"
)

func run() string {
	re := regexp.MustCompile(`\d+`)
	out := re.FindAllString("a1 b22 c333", -1)
	return strings.Join(out, ",")
}
