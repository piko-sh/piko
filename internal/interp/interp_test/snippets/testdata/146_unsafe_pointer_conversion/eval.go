package main

import "unsafe"

func run() int {
	x := 42
	p := unsafe.Pointer(&x)
	ip := (*int)(p)
	return *ip
}
