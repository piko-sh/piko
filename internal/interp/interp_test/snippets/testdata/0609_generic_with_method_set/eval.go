package main

type Named interface {
	Name() string
}

func greet[T Named](v T) string {
	return "hi " + v.Name()
}

type person struct {
	n string
}

func (p person) Name() string { return p.n }

func run() string {
	return greet(person{n: "Alice"})
}
