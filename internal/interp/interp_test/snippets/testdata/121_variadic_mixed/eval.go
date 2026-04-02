package main

func first(prefix string, nums ...int) int {
	r := len(prefix)
	for _, n := range nums {
		r += n
	}
	return r
}

func run() int {
	return first("abc", 10, 20)
}
