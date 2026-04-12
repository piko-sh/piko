package main

import "fmt"

func entrypoint() string {
	bump("one")
	bump("two")
	bump("three")
	return fmt.Sprintf("counter=%d history=%v", counter, history)
}

func main() {
	fmt.Println(entrypoint())
}
