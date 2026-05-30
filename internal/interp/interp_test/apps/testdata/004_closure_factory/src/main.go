package main

import "fmt"

func entrypoint() int {
	double := makeMultiplier(2)
	triple := makeMultiplier(3)
	addTen := makeAdder(10)

	_ = addTen(double(5))
	return double(10) + triple(10)
}

func main() {
	fmt.Println(entrypoint())
}
