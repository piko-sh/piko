package main

import (
	"fmt"
	"runtime"
)

func entrypoint() string {
	runtime.Goexit()
	return "unreachable"
}

func main() {
	fmt.Println(entrypoint())
}
