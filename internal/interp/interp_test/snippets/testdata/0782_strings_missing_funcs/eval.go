package main

import (
	"fmt"
	"strings"
)

func run() string {
	result := ""

	fields := strings.Fields("  hello   world  foo  ")
	result += fmt.Sprintf("fields=%v,len=%d;", fields, len(fields))

	result += fmt.Sprintf("containsAny:abc_in_lazy=%t,xyz_in_lazy=%t;",
		strings.ContainsAny("lazy fox", "abc"),
		strings.ContainsAny("lazy fox", "xyz"))

	before, after, found := strings.Cut("key=value", "=")
	result += fmt.Sprintf("cut=%s/%s,%t;", before, after, found)

	a, ok := strings.CutPrefix("prefix-data", "prefix-")
	result += fmt.Sprintf("cutpre=%s,%t;", a, ok)

	b, ok2 := strings.CutSuffix("data.txt", ".txt")
	result += fmt.Sprintf("cutsuf=%s,%t;", b, ok2)

	result += fmt.Sprintf("idxany=%d", strings.IndexAny("hello world", "wxyz"))
	return result
}
