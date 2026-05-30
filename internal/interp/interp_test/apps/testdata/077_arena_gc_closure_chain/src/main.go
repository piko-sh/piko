package main

import (
	"fmt"
	"strconv"
)

func makeProducer(prefix string) func(int) string {
	return func(n int) string {
		return prefix + strconv.Itoa(n)
	}
}

func entrypoint() string {
	producer := makeProducer("segment-")
	var final string
	callCount := 0
	for i := range 500 {
		final = producer(i)
		callCount++
	}
	return fmt.Sprintf("calls=%d final=%s", callCount, final)
}

func main() {
	fmt.Println(entrypoint())
}
