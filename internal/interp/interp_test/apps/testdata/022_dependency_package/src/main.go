package main

import (
	"fmt"

	"testpkg/mathkit"
)

func entrypoint() string {
	total := mathkit.Sum(1, 2, 3, 4, 5)
	doubled := mathkit.Double(total)
	squared := mathkit.Square(total)
	return fmt.Sprintf("sum=%d doubled=%d squared=%d", total, doubled, squared)
}

func main() {
	fmt.Println(entrypoint())
}
