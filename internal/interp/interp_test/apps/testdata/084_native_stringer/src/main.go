package main

import (
	"fmt"

	"testpkg/lib"
)

func entrypoint() string {
	return "tag=" + fmt.Sprintf("%s", lib.New("alpha"))
}

func main() {
	fmt.Println(entrypoint())
}
