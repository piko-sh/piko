package main

type Inner struct {
	V int
}

type Outer struct {
	Name  string
	Inner Inner
}

func run() int {
	m := map[string]*Outer{
		"first":  {Name: "a", Inner: Inner{V: 10}},
		"second": {Name: "b", Inner: Inner{V: 32}},
	}
	return m["first"].Inner.V + m["second"].Inner.V
}
