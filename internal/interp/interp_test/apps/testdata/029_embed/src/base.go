package main

type human struct {
	name string
}

func (h human) greet() string {
	return "hello-" + h.name
}
