package main

import (
	"fmt"
	"sync/atomic"
)

func entrypoint() string {
	var x int64
	atomic.AddInt64(&x, 1)
	return fmt.Sprintf("x=%d", x)
}

func main() {
	fmt.Println(entrypoint())
}
