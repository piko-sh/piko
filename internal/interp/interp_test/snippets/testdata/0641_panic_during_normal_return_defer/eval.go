package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	defer func() {
		panic("defer panic")
	}()
	return 99
}
