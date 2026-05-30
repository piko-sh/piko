package main

func run() int {
	a := 10
	b := 20
	c := 30
	a, b, c = c, a, b
	return a + b + c
}
