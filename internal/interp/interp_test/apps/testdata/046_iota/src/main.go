package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("small=%d large=%d lowDay=%d highDay=%d", small, large, mon, fri)
}

func main() {
	fmt.Println(entrypoint())
}
