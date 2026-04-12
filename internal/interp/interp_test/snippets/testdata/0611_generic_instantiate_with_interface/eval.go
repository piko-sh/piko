package main

type Box[T any] struct {
	v T
}

type Speaker interface {
	Say() string
}

type dog struct{}

func (dog) Say() string { return "woof" }

func run() string {
	b := Box[Speaker]{v: dog{}}
	return b.v.Say()
}
