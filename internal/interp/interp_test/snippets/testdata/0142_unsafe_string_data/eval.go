package main

import "unsafe"

func run() string {
	s := "world"
	ptr := unsafe.StringData(s)
	return unsafe.String(ptr, len(s))
}
