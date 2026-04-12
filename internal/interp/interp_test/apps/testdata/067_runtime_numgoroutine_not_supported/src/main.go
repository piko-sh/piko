package main

import (
	"fmt"
	"runtime"
)

func entrypoint() string {
	n := runtime.NumGoroutine()
	if n > 0 {
		return "ok"
	}
	return fmt.Sprintf("n=%d", n)
}

func main() {
	fmt.Println(entrypoint())
}
