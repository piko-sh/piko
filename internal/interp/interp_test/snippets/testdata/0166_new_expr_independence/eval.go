package main

func run() int {
	x := 42
	p := new(x)
	*p = 99
	return x
}
