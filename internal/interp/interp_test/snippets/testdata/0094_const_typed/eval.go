package main

const Pi float64 = 3.14159
const Greeting string = "hello"

func run() int {
	f := Pi * 2
	r := int(f)
	if Greeting == "hello" {
		r += 100
	}
	return r
}
