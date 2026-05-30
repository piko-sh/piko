package main

import "fmt"

func run() string {
	a := fmt.Sprintf("%T", 42)
	b := fmt.Sprintf("%T", "hi")
	c := fmt.Sprintf("%T", []int{1, 2})
	return a + "|" + b + "|" + c
}
