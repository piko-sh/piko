package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("a=%d b=%d c=%d", a, b, c)
}

func main() {
	fmt.Println(entrypoint())
}
