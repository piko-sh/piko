package main

func run() int {
	r := 0
	if 1.5 != 2.5 {
		r += 1
	}
	if 3.0 > 2.0 {
		r += 2
	}
	if 2.0 >= 2.0 {
		r += 4
	}
	if 3.0 >= 2.0 {
		r += 8
	}
	if !(1.0 != 1.0) {
		r += 16
	}
	return r
}
