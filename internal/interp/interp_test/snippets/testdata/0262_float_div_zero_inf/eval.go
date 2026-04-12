package main

func run() bool {
	x := 1.0
	y := 0.0
	z := x / y
	return z > 1e308
}
