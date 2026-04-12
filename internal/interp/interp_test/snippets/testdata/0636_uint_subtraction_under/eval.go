package main

func run() int {
	var x uint64 = 5
	var y uint64 = 10
	r := x - y
	if r > 1000 {
		return 1
	}
	return 0
}
