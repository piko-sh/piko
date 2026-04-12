package main

import "strings"

func run() string {
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString("x")
	}
	b.WriteByte('!')
	return b.String()
}
