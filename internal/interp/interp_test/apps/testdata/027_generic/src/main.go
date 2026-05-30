package main

import "fmt"

func entrypoint() string {
	a := maxOf(3, 9)
	b := twice(7)
	c := maxOf("apple", "pear")
	d := twice("zz")
	return fmt.Sprintf("a=%d b=%d c=%s d=%s", a, b, c, d)
}

func main() {
	fmt.Println(entrypoint())
}
