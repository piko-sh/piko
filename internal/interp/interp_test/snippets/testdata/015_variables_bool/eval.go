package main

func run() int {
	a := true
	b := false
	r := 0
	if !a {
		r += 1
	}
	if a && b {
		r += 2
	}
	if a || b {
		r += 4
	}
	if a != b {
		r += 8
	}
	if a == a {
		r += 16
	}
	return r
}
