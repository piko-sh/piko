package main

import "unsafe"

func run() int {
	return int(unsafe.Alignof(int64(0)))
}
