package main

import (
	"io"
	"strings"
)

func run() int {
	r := strings.NewReader("hi")
	buf := make([]byte, 10)
	_, _ = r.Read(buf)
	_, err := r.Read(buf)
	if err == io.EOF {
		return 1
	}
	return 0
}
