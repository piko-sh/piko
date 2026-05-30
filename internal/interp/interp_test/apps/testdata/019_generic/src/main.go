package main

import "fmt"

func entrypoint() string {
	v := identity(10)
	m := max2(v, 20)
	return fmt.Sprint(m)
}

func main() {
	fmt.Println(entrypoint())
}
