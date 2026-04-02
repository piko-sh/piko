package main

import "unsafe"

func run() int {
	arr := [3]int{11, 22, 33}
	ptr := unsafe.Pointer(&arr[0])
	ptr2 := unsafe.Add(ptr, 8)
	return *(*int)(ptr2)
}
