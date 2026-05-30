package main

type Animal struct {
	Name string
}

func (a Animal) breathe() string {
	return "ok"
}
