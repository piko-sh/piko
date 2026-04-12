package main

func run() int {
	a := [5]int{2, 4, 6, 8, 10}
	sum := 0
	for _, v := range a {
		sum += v
	}
	return sum
}
