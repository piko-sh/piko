package main

import (
	"fmt"
	"testpkg/lib"
)

func entrypoint() string {
	return lib.F(&lib.Holder{Name: "world"})
}

func main() {
	fmt.Println(entrypoint())
}
