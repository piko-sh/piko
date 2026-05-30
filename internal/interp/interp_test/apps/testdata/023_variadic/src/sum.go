package main

func sumAll(values ...int) int {
	total := 0
	for _, v := range values {
		total += v
	}
	return total
}
