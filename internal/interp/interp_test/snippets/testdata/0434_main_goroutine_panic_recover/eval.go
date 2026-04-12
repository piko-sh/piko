package main

func run() (result int) {
	defer func() {
		if r := recover(); r != nil {
			result = 11
		}
	}()
	panic("boom")
}
