package main

func run() int {
	sum := 0
	for i := range 10 {
		sum += i
	}
	return sum
}
