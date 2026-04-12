package main

type Box struct {
	N int
}

func (b Box) MutateLocal() int {
	b.N = 99
	return b.N
}

func run() int {
	p := &Box{N: 7}
	r := p.MutateLocal()
	return p.N*100 + r
}
