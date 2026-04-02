package main

import "unsafe"

func run() int {
	s := []int{100, 200, 300}
	ptr := unsafe.SliceData(s)
	return *ptr
}
