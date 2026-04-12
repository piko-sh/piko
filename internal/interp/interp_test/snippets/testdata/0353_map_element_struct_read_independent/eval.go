package main

type Box struct {
	N int
}

func run() int {
	m := map[string]Box{"a": {N: 7}}
	x := m["a"]
	x.N = 99
	return m["a"].N*100 + x.N
}
