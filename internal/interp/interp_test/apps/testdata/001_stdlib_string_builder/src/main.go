package main

import (
	"fmt"
	"strings"
)

func entrypoint() string {
	var b strings.Builder
	b.WriteString("<ul>")
	for _, item := range []string{"a", "b", "c"} {
		b.WriteString("<li>")
		b.WriteString(item)
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	return b.String()
}

func main() {
	fmt.Println(entrypoint())
}
