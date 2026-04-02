package main

type Entry struct {
	Name   string
	Invoke func(int) int
}

func run() int {
	registry := map[string]Entry{
		"double": {Name: "double", Invoke: func(x int) int { return x * 2 }},
	}
	return registry["double"].Invoke(21)
}
