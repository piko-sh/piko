package main

import "fmt"

type Inner struct {
	X int
}

type Outer struct {
	I Inner
	N string
}

func run() string {
	o := Outer{I: Inner{X: 7}, N: "outer"}
	return fmt.Sprintf("%+v", o)
}
