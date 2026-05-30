package main

func run() int {
	counter := 0
	bumper := func() {
		counter++
	}
	for k := 0; k < 5; k++ {
		bumper()
	}
	return counter
}
