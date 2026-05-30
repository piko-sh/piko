package main

type Big struct {
	A int
	B float64
	C string
	D bool
	E []int
	F map[string]int
}

func run() int {
	var b Big
	if b.A == 0 && b.B == 0.0 && b.C == "" && !b.D && b.E == nil && b.F == nil {
		return 1
	}
	return 0
}
