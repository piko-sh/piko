package main

type Acc struct {
	V int
}

func (a *Acc) Add(n int) *Acc {
	a.V += n
	return a
}

func run() int {
	a := &Acc{V: 0}
	a.Add(10).Add(20).Add(12)
	return a.V
}
