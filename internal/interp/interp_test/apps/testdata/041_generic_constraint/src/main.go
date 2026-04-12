package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("intMax=%d floatMax=%.2f strMax=%s",
		maxOf(3, 9), maxOf(3.14, 2.71), maxOf("apple", "zebra"))
}

func main() {
	fmt.Println(entrypoint())
}
