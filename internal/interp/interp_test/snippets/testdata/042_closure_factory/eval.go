package main

func makeAdder(base int) func(int) int {
	return func(n int) int {
		return base + n
	}
}

func run() int {
	add10 := makeAdder(10)
	add20 := makeAdder(20)
	return add10(5) + add20(5)
}
