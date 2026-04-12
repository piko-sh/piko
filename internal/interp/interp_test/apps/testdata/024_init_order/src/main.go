package main

import (
	"fmt"
	"strings"
)

var trail []string

func entrypoint() string {
	trail = append(trail, "entry")
	return strings.Join(trail, " ")
}

func main() {
	fmt.Println(entrypoint())
}
