package main

import "fmt"

func entrypoint() string {
	panic("boom")
}

func main() {
	fmt.Println(entrypoint())
}
