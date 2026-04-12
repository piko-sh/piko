package main

import "fmt"

func entrypoint() string {
	record("run")
	return fmt.Sprintf("events=%v count=%d", events, len(events))
}

func main() {
	fmt.Println(entrypoint())
}
