package main

import (
	"fmt"
	"testpkg/helpers"
)

func entrypoint() string {
	return helpers.Greet()
}

func main() {
	fmt.Println(entrypoint())
}
