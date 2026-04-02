package main

import "fmt"

func sum(nums []int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func entrypoint() int {
	data := []int{-3, -1, 0, 2, 5, -4, 8}
	// filterPositive: [2, 5, 8]
	// doubleAll:      [4, 10, 16]
	// sum:            30
	return sum(doubleAll(filterPositive(data)))
}

func main() {
	fmt.Println(entrypoint())
}
