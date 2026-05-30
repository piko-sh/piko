package main

type A struct{}

func (A) Tag() string { return "A" }

type B struct{ A }

type C struct{ B }

func run() string {
	c := C{}
	return c.Tag()
}
