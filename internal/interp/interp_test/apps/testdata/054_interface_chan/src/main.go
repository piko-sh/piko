package main

import (
	"fmt"
	"strings"
)

func entrypoint() string {
	ch := make(chan Event, 3)
	go produce(ch)
	var parts []string
	for ev := range ch {
		switch v := ev.(type) {
		case NumberEvent:
			parts = append(parts, fmt.Sprintf("got=number=%d", v.N))
		case TextEvent:
			parts = append(parts, fmt.Sprintf("got=text=%s", v.S))
		}
	}
	return strings.Join(parts, " ")
}

func main() {
	fmt.Println(entrypoint())
}
