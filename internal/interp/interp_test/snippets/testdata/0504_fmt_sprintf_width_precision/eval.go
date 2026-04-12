package main

import "fmt"

func run() string {
	a := fmt.Sprintf("%5d", 42)
	b := fmt.Sprintf("%-5d", 42)
	c := fmt.Sprintf("%.3f", 3.14159)
	d := fmt.Sprintf("%8.3f", 3.14159)
	return "[" + a + "][" + b + "][" + c + "][" + d + "]"
}
