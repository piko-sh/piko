package main

import (
	"fmt"
	"strings"
)

func panickyWork(target int) string {
	chunk := strings.Repeat("h", 200)
	for i := range target {
		_ = chunk + "x"
		if i == target-1 {
			panic("panic-payload-" + strings.Repeat("X", 5))
		}
	}
	return "unreachable"
}

func runWithRecover(iterations int) (recovered string, completed int) {
	defer func() {
		if v := recover(); v != nil {
			if s, ok := v.(string); ok {
				recovered = s
			} else {
				recovered = "non-string"
			}
		}
	}()
	_ = panickyWork(iterations)
	return "no-panic", iterations
}

func entrypoint() string {
	const iterations = 3000
	recovered, _ := runWithRecover(iterations)
	return fmt.Sprintf("recovered=%s iterations=%d", recovered, iterations)
}

func main() {
	fmt.Println(entrypoint())
}
