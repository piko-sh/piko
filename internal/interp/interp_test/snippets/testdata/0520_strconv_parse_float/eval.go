package main

import "strconv"

func run() int {
	v, err := strconv.ParseFloat("3.14", 64)
	if err != nil {
		return -1
	}
	if v > 3.1 && v < 3.2 {
		return 1
	}
	return 0
}
