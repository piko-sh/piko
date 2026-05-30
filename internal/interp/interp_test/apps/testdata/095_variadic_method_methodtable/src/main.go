package main

import (
	"fmt"
	"testpkg/lib"
)

func entrypoint() string {
	p := &lib.Printer{Separator: "|"}
	return p.Print("alpha", "beta", "gamma")
}

func main() {
	fmt.Println(entrypoint())
}
