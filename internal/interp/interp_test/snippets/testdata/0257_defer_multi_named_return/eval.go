package main

func helper() (a int, b int) {
	a = 10
	b = 100
	defer func() {
		a *= 2
		b *= 2
	}()
	return
}

func run() int {
	a, b := helper()
	return a + b
}
