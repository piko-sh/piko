package main

type Base struct {
	ID int
}
type Extended struct {
	Base
	Name string
}

func run() int {
	e := Extended{Base: Base{ID: 42}, Name: "test"}
	return e.ID
}
