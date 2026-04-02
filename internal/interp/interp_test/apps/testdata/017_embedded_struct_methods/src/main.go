package main

import "fmt"

func entrypoint() int {
	e := Extended{Base: Base{ID: 42}, Name: "test"}
	return e.ID + e.ID
}

func main() {
	fmt.Println(entrypoint())
}
