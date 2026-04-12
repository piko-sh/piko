package main

type Box struct {
	N int
}

func run() int {
	s := []Box{{N: 1}, {N: 2}, {N: 3}}
	for _, v := range s {
		v.N = 99
	}
	return s[0].N*100 + s[1].N*10 + s[2].N
}
