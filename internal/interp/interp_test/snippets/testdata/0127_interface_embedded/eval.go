package main

type Adder interface {
	Add(int)
}
type Getter interface {
	Get() int
}
type AdderGetter interface {
	Adder
	Getter
}
type Val struct {
	N int
}

func (v *Val) Add(n int) {
	v.N += n
}
func (v *Val) Get() int {
	return v.N
}
func use(ag AdderGetter) int {
	ag.Add(10)
	ag.Add(20)
	return ag.Get()
}

func run() int {
	v := &Val{N: 12}
	return use(v)
}
