package main

type Inner struct {
	Name string
}

type Outer struct {
	Inner
	Name string
}

func run() string {
	o := Outer{Inner: Inner{Name: "inner-name"}, Name: "outer-name"}
	return o.Name + "|" + o.Inner.Name
}
