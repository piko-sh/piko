package main

import "fmt"

type Speaker struct {
	name string
}

func (s Speaker) Greet() string { return "hi " + s.name }
func (s Speaker) Bye() string   { return "bye " + s.name }

func callGreet(x interface{ Greet() string }) string {
	return x.Greet()
}

func combo(x interface {
	Greet() string
	Bye() string
}) string {
	return x.Greet() + ";" + x.Bye()
}

func run() string {
	result := ""
	s := Speaker{name: "alice"}
	result += fmt.Sprintf("anon=%s;", callGreet(s))
	result += fmt.Sprintf("combo=%s", combo(s))
	return result
}
