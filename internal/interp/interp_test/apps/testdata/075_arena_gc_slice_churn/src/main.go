package main

import "fmt"

func entrypoint() string {
	var lastSlice []int
	sum := 0
	for iteration := range 2000 {
		buffer := make([]int, 100)
		for i := range buffer {
			buffer[i] = i
		}
		if iteration == 1999 {
			lastSlice = buffer
		}
	}
	for _, value := range lastSlice {
		sum += value
	}
	return fmt.Sprintf("sum=%d last=%d", sum, lastSlice[99])
}

func main() {
	fmt.Println(entrypoint())
}
