package main

func run() int {
	result := 0
	func() {
		defer func() { recover() }()
		panic("boom")
	}()
	result = 42
	return result
}
