package main

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func wrap(values any) []int {
	if typed, ok := values.([]int); ok {
		return typed
	}
	return nil
}

func run() int {
	directInts := []int{1, 2, 3, 4, 5}
	directSum := sumInts(directInts)

	wrapped := wrap(directInts)
	wrappedSum := sumInts(wrapped)

	return directSum + wrappedSum
}
