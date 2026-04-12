package main

import "fmt"

func entrypoint() string {
	total := sumAll(1, 2, 3, 4, 5)
	return fmt.Sprintf("sum=%d", total)
}

func main() {
	fmt.Println(entrypoint())
}
