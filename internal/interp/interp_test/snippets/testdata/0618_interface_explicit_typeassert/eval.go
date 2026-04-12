package main

type Speaker interface {
	Say() string
}

type Cat struct{}

func (Cat) Say() string { return "meow" }

func run() string {
	var s Speaker = Cat{}
	c, ok := s.(Cat)
	if !ok {
		return "miss"
	}
	return c.Say()
}
