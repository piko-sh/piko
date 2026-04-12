package main

import (
	"fmt"

	"testpkg/lib"
)

func entrypoint() string {
	buf := lib.NewBuffer()
	fmt.Fprint(buf, "fprint-here|")
	buf.Write([]byte("direct-here"))
	return "fp-buf=>>" + buf.String() + "<<"
}

func main() {
	fmt.Println(entrypoint())
}
