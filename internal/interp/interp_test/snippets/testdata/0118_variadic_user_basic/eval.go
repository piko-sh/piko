package main

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func run() int {
	return sum(10, 20, 12)
}
