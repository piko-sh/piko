package main

var u uint = 42

func run() int {
	n := int(u)
	u2 := uint(n + 8)
	return int(u2)
}
