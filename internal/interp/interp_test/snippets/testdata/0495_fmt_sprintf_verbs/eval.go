package main

import "fmt"

func run() string {
	a := fmt.Sprintf("%d", 42)
	b := fmt.Sprintf("%s", "hi")
	c := fmt.Sprintf("%v", true)
	d := fmt.Sprintf("%x", 255)
	e := fmt.Sprintf("%b", 5)
	f := fmt.Sprintf("%o", 8)
	return a + "|" + b + "|" + c + "|" + d + "|" + e + "|" + f
}
