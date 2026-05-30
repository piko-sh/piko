package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("red=%s green=%s blue=%s", hexCodes[Red], hexCodes[Green], hexCodes[Blue])
}

func main() {
	fmt.Println(entrypoint())
}
