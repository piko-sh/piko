package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	mode := flag.String("mode", "endtoend", "endtoend or inner")
	innerK := flag.Int("k", 0, "inner-loop iterations (when mode=inner)")
	flag.Parse()

	switch *mode {
	case "inner":
		result, elapsedNanos := RunInner(*innerK)
		fmt.Println(result)
		fmt.Fprintf(os.Stderr, "INNER_ELAPSED_NS=%d\n", elapsedNanos)
	default:
		fmt.Println(Run())
	}
}
