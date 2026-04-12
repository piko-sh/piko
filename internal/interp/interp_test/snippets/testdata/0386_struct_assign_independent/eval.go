package main

type Box struct {
	N int
}

func run() int {
	s1 := Box{N: 7}
	s2 := s1
	s2.N = 99
	return s1.N*100 + s2.N
}
