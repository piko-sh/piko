package main

import (
	"fmt"
	"testpkg/consts"
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
	return "pi=" + itoa(consts.Pi) + " e=" + itoa(consts.E) + " tau=" + itoa(consts.Tau)
}

func main() {
	fmt.Println(entrypoint())
}
