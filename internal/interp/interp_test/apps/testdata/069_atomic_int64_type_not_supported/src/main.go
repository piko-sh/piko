package main

import (
	"fmt"
	"sync/atomic"
)

func entrypoint() string {
	var x atomic.Int64
	x.Add(1)
	return fmt.Sprintf("x=%d", x.Load())
}

func main() {
	fmt.Println(entrypoint())
}
