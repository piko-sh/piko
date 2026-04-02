package main

import (
	"fmt"
	"testpkg/counter"
)

func entrypoint() string {
	c := counter.New()
	c.Increment()
	c.Increment()
	c.Increment()
	return fmt.Sprintf("counter:%d", c.Value())
}

func main() {
	fmt.Println(entrypoint())
}
