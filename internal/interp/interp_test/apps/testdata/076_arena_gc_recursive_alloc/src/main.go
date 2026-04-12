package main

import (
	"fmt"
	"strings"
)

func accumulate(depth int, total int) int {
	if depth == 0 {
		return total
	}
	chunk := strings.Repeat("z", depth*50)
	return accumulate(depth-1, total+len(chunk))
}

func entrypoint() string {
	result := accumulate(200, 0)
	return fmt.Sprintf("depth=200 builtLen=%d", result)
}

func main() {
	fmt.Println(entrypoint())
}
