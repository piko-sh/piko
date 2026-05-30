package main

import "fmt"

func entrypoint() string {
	ch := make(chan int, 1)
	go func() {
		ch <- 1
	}()
	panic("main panic")
}

func main() {
	fmt.Println(entrypoint())
}
