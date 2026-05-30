package main

type Greeter interface {
	Hi() string
}

type Casual struct{ Name string }

func (c Casual) Hi() string { return "hi-" + c.Name }

func run() string {
	var g Greeter = Casual{Name: "x"}
	c, ok := g.(Casual)
	if !ok {
		return "fail"
	}
	return c.Name
}
