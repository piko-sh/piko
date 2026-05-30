package main

type Box struct {
	N int
}

func consume(v interface{}) int {
	if box, ok := v.(Box); ok {
		box.N = 99
		return box.N
	}
	return -1
}

func run() int {
	s := Box{N: 7}
	r := consume(s)
	return s.N*1000 + r
}
