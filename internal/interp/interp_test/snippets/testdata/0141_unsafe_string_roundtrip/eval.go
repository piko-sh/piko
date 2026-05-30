package main

import "unsafe"

func run() string {
	buffer := []byte("hello")
	s := unsafe.String(&buffer[0], len(buffer))
	return s
}
