package main

import (
	"fmt"

	"testpkg/lib"
)

func entrypoint() string {
	buf := lib.NewBuffer()
	buf.Write([]byte("hello world"))
	return "buf=>>" + buf.String() + "<<"
}

func main() {
	fmt.Println(entrypoint())
}
