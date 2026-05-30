package main

import (
	"fmt"

	"testpkg/parsed"
)

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune(48+n%10)) + s
		n = n / 10
	}
	return s
}

func entrypoint() string {
	return "greeting=" + parsed.Greeting + " len=" + itoa(parsed.Length)
}

func main() {
	fmt.Println(entrypoint())
}
