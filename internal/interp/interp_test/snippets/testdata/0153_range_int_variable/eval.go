package main

func run() int {
	n := 7
	s := 0
	for i := range n {
		s += i
	}
	return s
}
