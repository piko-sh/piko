package main

import "fmt"

func entrypoint() string {
	got := runSafely()
	return fmt.Sprintf("recovered=%s finished=%t", got, true)
}

func main() {
	fmt.Println(entrypoint())
}
