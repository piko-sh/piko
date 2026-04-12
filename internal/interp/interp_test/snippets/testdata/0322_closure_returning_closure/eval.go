package main

func adder(n int) func(int) func(int) int {
	return func(m int) func(int) int {
		return func(k int) int {
			return n + m + k
		}
	}
}

func run() int {
	a := adder(1)
	b := a(2)
	return b(3) + b(4)
}
