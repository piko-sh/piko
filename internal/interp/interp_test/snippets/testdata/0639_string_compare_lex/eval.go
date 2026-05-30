package main

func run() int {
	if "abc" < "abd" && "abc" < "abca" {
		return 1
	}
	return 0
}
