package main

import (
	"fmt"

	"testpkg/lib"
)

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	s := ""
	for n > 0 {
		s = string(rune(48+n%10)) + s
		n = n / 10
	}
	if negative {
		s = "-" + s
	}
	return s
}

func entrypoint() string {
	return "answer=" + itoa(lib.Answer) + " doubled=" + itoa(lib.Answer*2)
}

func main() {
	fmt.Println(entrypoint())
}
