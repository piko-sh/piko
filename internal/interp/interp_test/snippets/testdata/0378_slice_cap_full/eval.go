package main

func run() int {
	s := make([]int, 3, 10)
	s[0], s[1], s[2] = 1, 2, 3
	full := s[:cap(s)]
	full[5] = 999
	if len(full) != 10 {
		return -1
	}
	return s[0] + full[5]
}
