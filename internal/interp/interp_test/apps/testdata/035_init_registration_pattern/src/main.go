package main

import "fmt"

func entrypoint() string {
	return fmt.Sprintf("json=[%s] yaml=[%s] xml=[%s] count=%d",
		dispatch("json"), dispatch("yaml"), dispatch("xml"), len(handlers))
}

func main() {
	fmt.Println(entrypoint())
}
