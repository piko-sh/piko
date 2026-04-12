package main

type Speaker interface {
	Say() string
}

type Greeter struct {
	Phrase string
}

func (g *Greeter) Say() string {
	return g.Phrase + "!"
}

func makeIt() Speaker {
	return &Greeter{Phrase: "hi"}
}

func run() string {
	s := makeIt()
	return s.Say()
}
