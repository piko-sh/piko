package main

import "unsafe"

type Point struct {
	X int
	Y int
}

func run() int {
	return int(unsafe.Sizeof(Point{}))
}
