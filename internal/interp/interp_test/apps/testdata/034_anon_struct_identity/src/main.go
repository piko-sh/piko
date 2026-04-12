package main

import "fmt"

func entrypoint() string {
	p := produce()
	name, age := consume(p)
	match := name == "Bob" && age == 30
	return fmt.Sprintf("match=%t name=%s age=%d", match, name, age)
}

func main() {
	fmt.Println(entrypoint())
}
