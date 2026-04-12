package main

func run() int {
	total := 0
	for i := 0; i < 6; i++ {
		total += i*7 + 3
	}
	return total
}
