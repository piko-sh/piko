package main

import "unsafe"

type Pair struct {
	A int32
	B int64
}

func run() int {
	return int(unsafe.Offsetof(Pair{}.B))
}
