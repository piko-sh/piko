package main

func leaf(log []int) {
	defer record(log, 2)
	log[0]++
}

func record(log []int, value int) {
	log[1] = value
}

func outer(log []int) {
	defer record(log, 1)
	leaf(log)
}

func run() int {
	log := make([]int, 2)
	outer(log)
	return log[0]*100 + log[1]
}
