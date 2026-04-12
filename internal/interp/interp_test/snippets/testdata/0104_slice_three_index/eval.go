package main

func run() int {
	s := []int{1, 2, 3, 4, 5}
	t := s[1:3:4]
	return len(t)*10 + cap(t)
}
