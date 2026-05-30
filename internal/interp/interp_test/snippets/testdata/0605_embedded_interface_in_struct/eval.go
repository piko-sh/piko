package main

type Greeter interface {
	Greet() string
}

type Polite struct{}

func (Polite) Greet() string { return "hello!" }

type Wrapped struct {
	Greeter
}

func run() string {
	w := Wrapped{Greeter: Polite{}}
	return w.Greet()
}
