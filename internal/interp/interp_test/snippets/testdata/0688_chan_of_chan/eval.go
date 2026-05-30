package main

import "fmt"

func run() string {
	outer := make(chan chan int, 1)
	inner := make(chan int, 1)

	inner <- 42
	outer <- inner

	receivedChan := <-outer
	value := <-receivedChan
	return fmt.Sprintf("value=%d", value)
}
