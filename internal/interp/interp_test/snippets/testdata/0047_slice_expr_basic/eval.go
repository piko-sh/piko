package main

func run() int {
	s := []int{10, 20, 30, 40, 50}
	t := s[1:4]
	return t[0] + t[1] + t[2]
}
