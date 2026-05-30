package main

import "fmt"

type Point struct {
	X, Y int
}

func run() string {
	p := Point{X: 1, Y: 2}
	return fmt.Sprintf("%v", p)
}
