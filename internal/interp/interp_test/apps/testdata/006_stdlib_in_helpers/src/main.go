package main

import "fmt"

func entrypoint() string {
	items := []string{"alpha", "beta", "gamma"}
	wrapped := make([]string, len(items))
	for i, item := range items {
		wrapped[i] = wrapTag("li", item)
	}
	return wrapTag("ul", joinAll(wrapped))
}

func main() {
	fmt.Println(entrypoint())
}
