package main

import "fmt"

var result = computeBase()
var doubled = computeDoubled(result)

func entrypoint() string {
	return fmt.Sprintf("result=%d doubled=%d", result, doubled)
}

func main() {
	fmt.Println(entrypoint())
}
