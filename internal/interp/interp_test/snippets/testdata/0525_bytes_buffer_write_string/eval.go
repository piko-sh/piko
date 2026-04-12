package main

import "bytes"

func run() string {
	var b bytes.Buffer
	b.WriteString("hello, ")
	b.WriteString("world")
	return b.String()
}
