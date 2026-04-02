package main

var a uint = 10
var b uint = 20

func run() int {
	r := 0
	if a < b {
		r += 1
	}
	if a <= b {
		r += 2
	}
	if b > a {
		r += 4
	}
	if b >= a {
		r += 8
	}
	if a == a {
		r += 16
	}
	if a != b {
		r += 32
	}
	return r
}
