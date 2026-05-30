package main

import (
	"fmt"
	"strings"
)

type Tagged struct {
	Name string
	Val  int
}

func run() string {
	result := ""

	t := Tagged{Name: "alpha", Val: 7}
	result += fmt.Sprintf("hash=%#v;", t)

	x := 42
	pStr := fmt.Sprintf("%p", &x)
	result += fmt.Sprintf("p_starts_0x=%t,p_len_ge_3=%t;",
		strings.HasPrefix(pStr, "0x"), len(pStr) >= 3)

	result += fmt.Sprintf("c=%c;", 'A')

	result += fmt.Sprintf("U=%U", rune(0x4E2D))
	return result
}
