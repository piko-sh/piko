package main

func doubleAll(nums []int) []int {
	result := make([]int, len(nums))
	for i, n := range nums {
		result[i] = n * 2
	}
	return result
}
