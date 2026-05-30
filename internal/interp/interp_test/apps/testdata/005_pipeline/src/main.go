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

	return sum(doubleAll(filterPositive(data)))
}

func main() {
	fmt.Println(entrypoint())
}
