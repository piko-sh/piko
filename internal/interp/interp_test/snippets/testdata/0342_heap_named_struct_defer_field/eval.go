package main

type Outcome struct {
	A int
	B int
}

func compute() (result Outcome) {
	defer func() {
		result.A = 100
		result.B = result.B + 50
	}()
	result.A = 1
	result.B = 2
	return
}

func run() int {
	r := compute()
	return r.A*1000 + r.B
}
