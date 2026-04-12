package main

import (
	"bufio"
	"strings"
)

func run() int {
	r := strings.NewReader("a\nb\nc\n")
	s := bufio.NewScanner(r)
	count := 0
	for s.Scan() {
		count++
	}
	return count
}
