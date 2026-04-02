package main

func run() bool {
	var isEven, isOdd func(int) bool
	isEven = func(n int) bool {
		if n == 0 {
			return true
		}
		return isOdd(n - 1)
	}
	isOdd = func(n int) bool {
		if n == 0 {
			return false
		}
		return isEven(n - 1)
	}
	return isEven(10)
}
