package main

import "unsafe"

func run() int {
	s := []int{10, 20, 30}
	ptr := unsafe.SliceData(s)
	s2 := unsafe.Slice(ptr, 3)
	str := "hello"
	sp := unsafe.StringData(str)
	str2 := unsafe.String(sp, len(str))
	return s2[0] + s2[1] + s2[2] + len(str2)
}
