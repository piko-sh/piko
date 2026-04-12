package main

import "unsafe"

func run() float64 {
	x := uint64(4607182418800017408)
	f := *(*float64)(unsafe.Pointer(&x))
	return f
}
