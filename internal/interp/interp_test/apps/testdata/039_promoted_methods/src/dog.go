package main

type Dog struct {
	Animal
}

func (d Dog) kind() string {
	return "dog"
}
