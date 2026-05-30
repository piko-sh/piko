package main

import "fmt"

func run() string {
	return fmt.Sprintf("a=%[1]s,b=%[2]d,a_again=%[1]s,b_dec=%[2]d,both=%[2]d/%[1]s",
		"hello", 42)
}
