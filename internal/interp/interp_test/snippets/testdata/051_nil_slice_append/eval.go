package main

var s []int

func run() int {
	s = append(s, 10)
	s = append(s, 20)
	s = append(s, 30)
	return s[0] + s[1] + s[2]
}
