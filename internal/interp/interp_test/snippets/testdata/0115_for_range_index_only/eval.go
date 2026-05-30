package main

func run() int {
	s := []int{10, 20, 30}
	sum := 0
	for i := range s {
		sum += i
	}
	return sum
}
