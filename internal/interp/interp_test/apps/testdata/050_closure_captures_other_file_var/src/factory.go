package main

func makeAdder(by int) func() {
	return func() {
		sharedTotal += by
	}
}
