package main

type Box struct{ V int }

func (b Box) Get() int { return b.V }

func run() int {
	b := Box{V: 10}
	f := b.Get
	b.V = 99
	return f()
}
