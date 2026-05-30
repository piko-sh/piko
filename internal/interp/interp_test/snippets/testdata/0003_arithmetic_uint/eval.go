package main

var a uint = 15
var b uint = 4

func run() int {
	return int(a*b + a/b + a%b)
}
