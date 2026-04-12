package main

var u uint = 7

func run() int {
	f := float64(u)
	u2 := uint(f * 3.0)
	return int(u2)
}
