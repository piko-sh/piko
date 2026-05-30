package main

type Box struct {
	N int
}

func run() int {
	s := []Box{{N: 1}, {N: 2}}
	x := s[0]
	x.N = 99
	return s[0].N*100 + x.N
}
