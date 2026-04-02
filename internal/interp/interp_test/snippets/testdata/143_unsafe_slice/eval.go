package main

import "unsafe"

func run() int {
	arr := [5]int{10, 20, 30, 40, 50}
	s := unsafe.Slice(&arr[0], 3)
	return s[0] + s[1] + s[2]
}
