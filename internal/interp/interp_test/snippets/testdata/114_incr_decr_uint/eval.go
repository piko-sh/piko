package main

var x uint = 10

func run() int {
	x++
	x++
	x--
	return int(x)
}
