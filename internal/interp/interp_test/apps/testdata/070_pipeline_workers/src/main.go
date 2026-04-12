package main

import "fmt"

func entrypoint() string {
	out := runPipeline(5)
	return fmt.Sprintf("sum=%d", out)
}

func main() {
	fmt.Println(entrypoint())
}
