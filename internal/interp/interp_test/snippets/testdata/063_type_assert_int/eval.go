package main

var x interface{} = 42

func run() int {
	v := x.(int)
	return v
}
