package main

var a [3]int

func run() int {
	a[0] = 10
	a[1] = 20
	a[2] = 30
	return a[0] + a[1] + a[2]
}
