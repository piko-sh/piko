package main

import (
	"fmt"
	"testpkg/middle"
)

func entrypoint() string {
	return middle.Wrap(middle.GetCore())
}

func main() {
	fmt.Println(entrypoint())
}
