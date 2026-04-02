package main

var x uint = 10

func run() int {
	x += 5
	x *= 2
	x -= 7
	x /= 3
	x %= 4
	return int(x)
}
