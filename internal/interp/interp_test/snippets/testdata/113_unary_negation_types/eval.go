package main

func run() int {
	a := -42
	b := -3.14
	c := -1 - 2i
	r := a
	if b < 0 {
		r += 100
	}
	if c == -1-2i {
		r += 1000
	}
	return r
}
