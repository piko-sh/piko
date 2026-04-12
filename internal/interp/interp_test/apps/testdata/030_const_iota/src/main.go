package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("small=%d medium=%d large=%d label=%s", small, medium, large, sizeLabel(medium))
}

func main() {
	fmt.Println(entrypoint())
}
