package main

import "bytes"

func run() int {
	a := []byte("hello")
	b := []byte("hello")
	if bytes.Equal(a, b) && bytes.Contains(a, []byte("ell")) {
		return 1
	}
	return 0
}
