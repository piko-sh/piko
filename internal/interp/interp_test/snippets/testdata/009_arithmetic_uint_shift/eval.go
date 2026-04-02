package main

var x uint = 1

func run() int {
	x = x << 10
	x = x >> 3
	return int(x)
}
