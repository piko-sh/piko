package main

func filterPositive(nums []int) []int {
	var result []int
	for _, n := range nums {
		if n > 0 {
			result = append(result, n)
		}
	}
	return result
}
