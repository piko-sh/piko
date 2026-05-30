package main

func run() int {
	original := []int{1, 2, 3}
	alias := original
	alias[0] = 99
	return original[0]
}
