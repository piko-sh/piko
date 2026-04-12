package main

import (
	"fmt"
	"testpkg/ops"
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
	adder := ops.MakeAdder(3)
	multiplier := ops.MakeMultiplier(5)
	return "add:" + itoa(adder(5)) + " mul:" + itoa(multiplier(3))
}

func main() {
	fmt.Println(entrypoint())
}
