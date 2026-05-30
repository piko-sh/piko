package main

import "fmt"

func entrypoint() string {
	b := makeBuffer()
	b.Append(1)
	b.Append(2)
	b.Append(3)
	directSum := b.Sum()

	buffers := makeBuffersInSlice(3)
	for i := range buffers {
		buffers[i].Append(10)
	}
	sliceSum := 0
	for i := range buffers {
		sliceSum += buffers[i].Sum()
	}

	return fmt.Sprintf("direct=%d slice=%d", directSum, sliceSum)
}

func main() {
	fmt.Println(entrypoint())
}
