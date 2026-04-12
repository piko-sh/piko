package main

type Animal interface {
	Sound() string
}

type Dog struct{}

func (Dog) Sound() string { return "woof" }

type Cat struct{}

func (Cat) Sound() string { return "meow" }

func run() string {
	zoo := []Animal{Dog{}, Cat{}, Dog{}}
	out := ""
	for _, a := range zoo {
		out += a.Sound()
	}
	return out
}
