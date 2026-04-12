package main

import (
	"bufio"
	"strings"
)

func run() string {
	r := bufio.NewReader(strings.NewReader("hello"))
	buf := make([]byte, 5)
	n, _ := r.Read(buf)
	return string(buf[:n])
}
