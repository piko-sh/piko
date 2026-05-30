package main

import (
	"fmt"
	"strings"
)

func run() string {
	s := fmt.Sprintln("a", 1, true, 2.5)
	return strings.TrimRight(s, "\n")
}
