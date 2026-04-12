package main

import "fmt"

func entrypoint() string {
	b := &Box[int]{Value: 10}
	b.Set(42)
	return fmt.Sprint(b.Get())
}

func main() {
	fmt.Println(entrypoint())
}
