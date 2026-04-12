package main

func tryRecover() any {
	return recover()
}

func run() (result int) {
	defer func() {
		nestedR := tryRecover()
		directR := recover()
		if nestedR == nil && directR != nil {
			result = 1
		} else {
			result = 0
		}
	}()
	panic("x")
}
