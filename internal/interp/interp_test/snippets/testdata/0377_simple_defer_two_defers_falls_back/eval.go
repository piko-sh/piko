package main

func run() int {
	s := []int{0}
	defer func() { s[0] += 10 }()
	defer func() { s[0] += 1 }()
	return s[0]
}
