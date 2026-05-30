package main

import "fmt"

func entrypoint() string {
	c := &counter{label: "hi"}
	c.bump()
	c.bump()
	c.bump()
	return fmt.Sprintf("label=%s count=%d doubled=%d", c.describe(), c.value, c.doubled())
}

func main() {
	fmt.Println(entrypoint())
}
