package main

import "fmt"

func entrypoint() string {
	g := newGreeter("hi")
	first := g.greet("")
	second := g.greet("")
	return fmt.Sprintf("first=%s second=%s count=%d", first, second, g.count)
}

func main() {
	fmt.Println(entrypoint())
}
