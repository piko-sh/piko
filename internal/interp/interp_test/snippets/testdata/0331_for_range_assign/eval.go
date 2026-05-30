package main

func run() int {
	xs := []int{10, 20, 30}
	var i int
	var v int
	total := 0
	for i, v = range xs {
		total += i*100 + v
	}
	return total
}
