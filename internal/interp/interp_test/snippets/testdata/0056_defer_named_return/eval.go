package main

func f() (result int) {
	defer func() {
		result = 42
	}()
	return 0
}

func run() int {
	return f()
}
