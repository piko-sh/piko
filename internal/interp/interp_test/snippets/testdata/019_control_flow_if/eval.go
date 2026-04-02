package main

func run() int {
	x := 15
	result := 0
	if x > 10 {
		if x > 20 {
			result = 3
		} else {
			result = 2
		}
	} else {
		result = 1
	}
	return result
}
