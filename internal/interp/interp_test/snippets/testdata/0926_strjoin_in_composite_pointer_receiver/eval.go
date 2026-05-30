package main

import "strings"

type myT struct{ name string }

func getStrings() []string {
	return []string{"hello", "world"}
}

func test(t *myT, msg string) string {
	content := []string{
		strings.Join(getStrings(), "-"),
		msg,
	}
	return content[0]
}

func run() int {
	r := test(&myT{name: "x"}, "fail")
	return len(r)
}
