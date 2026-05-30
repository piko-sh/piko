package main

import (
	"bytes"
	"io"
)

func run() string {
	src := bytes.NewBufferString("hello, copy!")
	var dst bytes.Buffer
	_, _ = io.Copy(&dst, src)
	return dst.String()
}
