package main

import (
	"fmt"
	"testpkg/pkgb"
	"testpkg/pkgc"
)

func entrypoint() string {
	return "B:" + pkgb.FromB() + " C:" + pkgc.FromC()
}

func main() {
	fmt.Println(entrypoint())
}
