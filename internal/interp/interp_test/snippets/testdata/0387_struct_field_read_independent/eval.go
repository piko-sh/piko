package main

type Inner struct {
	N int
}

type Outer struct {
	In Inner
}

func run() int {
	o := Outer{In: Inner{N: 7}}
	x := o.In
	x.N = 99
	return o.In.N*100 + x.N
}
