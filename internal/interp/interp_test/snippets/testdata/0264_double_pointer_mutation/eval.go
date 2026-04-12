package main

func run() int {
	x := 1
	p := &x
	pp := &p
	**pp = 99
	return x
}
