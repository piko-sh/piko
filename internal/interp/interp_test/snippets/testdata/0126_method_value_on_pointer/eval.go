package main

type Box struct {
	V int
}

func (b Box) Get() int {
	return b.V
}

func run() int {
	p := &Box{V: 42}
	return p.Get()
}
