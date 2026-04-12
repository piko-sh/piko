package main

import (
	"fmt"
	"testpkg/lib"
)

func entrypoint() string {
	return fmt.Sprintf("%d", lib.F("hello"))
}

func main() {
	fmt.Println(entrypoint())
}
