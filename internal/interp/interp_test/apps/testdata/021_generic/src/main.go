package main

import (
	"fmt"
	"testpkg/helpers"
)

func entrypoint() string {
	v := helpers.Identity(42)
	return fmt.Sprint(v)
}

func main() {
	fmt.Println(entrypoint())
}
