package main

func run() int {
	var x any = 42
	return x.(int) + 1
}
