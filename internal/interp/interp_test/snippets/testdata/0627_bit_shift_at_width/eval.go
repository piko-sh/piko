package main

func run() int {
	var x uint32 = 1
	r := x << 31
	return int(r)
}
