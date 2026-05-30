package main

import (
	"fmt"
	"time"
)

func entrypoint() string {
	go func() {
		panic("goroutine died")
	}()
	time.Sleep(20 * time.Millisecond)
	return "main continued"
}

func main() {
	fmt.Println(entrypoint())
}
