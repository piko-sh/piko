package main

import "fmt"

func entrypoint() string {
	n := Numbers{values: []int{10, 20, 30}}
	return fmt.Sprintf("len=%d first=%d sum=%d", n.Len(), n.First(), n.Sum())
}

func main() {
	fmt.Println(entrypoint())
}
