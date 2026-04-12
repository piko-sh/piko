package main

import "fmt"

func entrypoint() string {
	c := &counter{}
	launchAdders(c, 10, 3)
	return fmt.Sprintf("total=%d", c.value())
}

func main() {
	fmt.Println(entrypoint())
}
