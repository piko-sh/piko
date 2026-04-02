package main

import "fmt"

func entrypoint() string {
	return buildTag("h1", "Hello")
}

func main() {
	fmt.Println(entrypoint())
}
