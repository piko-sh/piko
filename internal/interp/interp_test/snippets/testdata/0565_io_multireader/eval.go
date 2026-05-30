package main

import (
	"bytes"
	"io"
	"strings"
)

func run() string {
	r := io.MultiReader(
		strings.NewReader("hello, "),
		strings.NewReader("world"),
	)
	var dst bytes.Buffer
	_, _ = io.Copy(&dst, r)
	return dst.String()
}
