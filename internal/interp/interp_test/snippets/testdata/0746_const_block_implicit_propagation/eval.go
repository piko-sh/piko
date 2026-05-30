package main

import "fmt"

const (
	A = 1
	B
	C = 5
	D
)

const (
	X = "hello"
	Y
	Z = "world"
	W
)

func run() string {
	return fmt.Sprintf("A=%d,B=%d,C=%d,D=%d;X=%s,Y=%s,Z=%s,W=%s", A, B, C, D, X, Y, Z, W)
}
