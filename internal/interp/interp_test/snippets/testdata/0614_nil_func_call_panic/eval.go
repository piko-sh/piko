package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 1
		}
	}()
	var f func()
	f()
	return 0
}
