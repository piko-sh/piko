package main

import "fmt"

func entrypoint() string {
	g := Greeter{Prefix: "Hi"}
	fn := g.Greet
	return fmt.Sprintf("first=%s second=%s third=%s", fn("Alice"), fn("Bob"), fn("Cara"))
}

func main() {
	fmt.Println(entrypoint())
}
