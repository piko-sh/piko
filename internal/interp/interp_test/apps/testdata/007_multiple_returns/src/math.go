package main

func divmod(a int, b int) (int, int) {
	return a / b, a % b
}

func safeDivide(a int, b int) (int, bool) {
	if b == 0 {
		return 0, false
	}
	return a / b, true
}
