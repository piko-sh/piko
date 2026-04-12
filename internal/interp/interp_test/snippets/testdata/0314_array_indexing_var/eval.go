package main

func run() int {
	var arr [5]int
	for i := range arr {
		arr[i] = i * i
	}
	idx := 3
	return arr[idx] + arr[len(arr)-1]
}
