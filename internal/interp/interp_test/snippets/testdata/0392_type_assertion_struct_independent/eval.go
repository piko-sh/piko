package main

type Box struct {
	N int
}

func run() int {
	var x interface{} = Box{N: 7}
	y, ok := x.(Box)
	if !ok {
		return -1
	}
	y.N = 99
	z, _ := x.(Box)
	return z.N*100 + y.N
}
