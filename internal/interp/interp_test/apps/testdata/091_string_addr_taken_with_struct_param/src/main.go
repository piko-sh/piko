package main

import "fmt"

type Holder struct {
	Name string
}

func f(s string, p *Holder) string {
	_ = &s
	return p.Name
}

func entrypoint() string {
	return f("hello", &Holder{Name: "world"})
}

func main() {
	fmt.Println(entrypoint())
}
