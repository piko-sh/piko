package main

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func run() int {
	s := []int{1, 2, 3, 4, 5}
	return sum(s...)
}
