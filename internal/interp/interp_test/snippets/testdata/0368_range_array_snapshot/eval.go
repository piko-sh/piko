package main

func run() int {
	arr := [3]int{1, 2, 3}
	sum := 0
	for i, v := range arr {
		if i == 0 {
			arr[1] = 99
			arr[2] = 99
		}
		sum += v
	}
	return sum
}
