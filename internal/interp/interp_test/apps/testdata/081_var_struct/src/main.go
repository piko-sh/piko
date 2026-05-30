package main

import (
	"fmt"

	"testpkg/data"
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
	p := data.Default
	return "name=" + p.Name + " year=" + itoa(p.Year)
}

func main() {
	fmt.Println(entrypoint())
}
