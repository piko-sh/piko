package main

import (
	"bytes"
	"io"
	"strings"
)

func run() string {
	r := io.LimitReader(strings.NewReader("hello, world"), 5)
	var dst bytes.Buffer
	_, _ = io.Copy(&dst, r)
	return dst.String()
}
