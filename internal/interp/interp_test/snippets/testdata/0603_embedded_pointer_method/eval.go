package main

type Inner struct {
	X int
}

func (i *Inner) Inc() {
	i.X++
}

type Outer struct {
	*Inner
}

func run() int {
	o := Outer{Inner: &Inner{X: 5}}
	o.Inc()
	o.Inc()
	return o.X
}
