package main

import (
	"fmt"
	"strings"
)

func entrypoint() string {
	chunk := strings.Repeat("X", 200)
	total := 0
	for range 5000 {
		total += len(chunk)
		_ = chunk + "extra"
	}
	finalChunk := strings.Repeat("Y", 5)
	return fmt.Sprintf("final_len=%d last5=%s", total, finalChunk)
}

func main() {
	fmt.Println(entrypoint())
}
