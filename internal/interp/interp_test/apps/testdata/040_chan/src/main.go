package main

import "fmt"

func entrypoint() string {
	ch := make(chan int, 5)
	go produceNumbers(ch)
	var received []int
	total := 0
	for v := range ch {
		received = append(received, v)
		total += v
	}
	return fmt.Sprintf("received=%v sum=%d", received, total)
}

func main() {
	fmt.Println(entrypoint())
}
