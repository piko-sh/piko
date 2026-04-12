package main

import (
	"bufio"
	"strings"
)

func run() string {
	r := strings.NewReader("alpha beta gamma")
	s := bufio.NewScanner(r)
	s.Split(bufio.ScanWords)
	parts := []string{}
	for s.Scan() {
		parts = append(parts, s.Text())
	}
	return strings.Join(parts, "|")
}
