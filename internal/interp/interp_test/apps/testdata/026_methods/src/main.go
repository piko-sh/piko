package main

import "fmt"

func entrypoint() string {
	p := person{name: "ada", age: 37}
	return fmt.Sprintf("name=%s age=%d greet=%s label=%s", p.name, p.age, p.greet(), p.label())
}

func main() {
	fmt.Println(entrypoint())
}
