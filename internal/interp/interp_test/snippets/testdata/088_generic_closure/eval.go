package main

func makeMultiplier[T ~int](factor T) func(T) T {
	return func(v T) T {
		return v * factor
	}
}

func run() int {
	triple := makeMultiplier(3)
	return triple(7)
}
