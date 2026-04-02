package main

func isEven(n int) bool {
	if n == 0 {
		return true
	}
	return isOdd(n - 1)
}
