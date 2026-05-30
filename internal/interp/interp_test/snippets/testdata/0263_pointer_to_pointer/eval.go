package main

func run() int {
	x := 42
	p := &x
	pp := &p
	return **pp
}
