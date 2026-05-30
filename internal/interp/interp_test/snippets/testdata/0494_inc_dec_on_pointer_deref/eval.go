package main

func run() int {
	x := 5
	p := &x
	*p++
	*p++
	*p--
	return *p
}
