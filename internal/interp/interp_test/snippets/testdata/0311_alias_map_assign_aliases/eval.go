package main

func run() int {
	original := map[string]int{"k": 7}
	alias := original
	alias["k"] = 99
	return original["k"]
}
