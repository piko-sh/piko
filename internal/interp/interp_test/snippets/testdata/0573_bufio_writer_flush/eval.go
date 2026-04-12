package main

import (
	"bufio"
	"bytes"
)

func run() string {
	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	_, _ = w.WriteString("hello")
	_ = w.Flush()
	return buf.String()
}
