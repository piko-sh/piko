package main

func run() int {
	z1 := 2 + 3i
	z2 := 2 + 3i
	z3 := 1 + 1i
	r := 0
	if z1 == z2 {
		r += 1
	}
	if z1 != z3 {
		r += 2
	}
	return r
}
