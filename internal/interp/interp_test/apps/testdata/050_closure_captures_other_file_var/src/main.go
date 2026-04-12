package main

import "fmt"

func entrypoint() string {
	before := sharedTotal
	add := makeAdder(2)
	add()
	add()
	return fmt.Sprintf("before=%d after=%d", before, sharedTotal)
}

func main() {
	fmt.Println(entrypoint())
}
