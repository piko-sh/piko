package main

import "fmt"

func entrypoint() string {
	e := employee{human: human{name: "Bob"}, role: "Engineer"}
	return fmt.Sprintf("name=%s role=%s greet=%s role=%s", e.name, e.role, e.greet(), e.jobTag())
}

func main() {
	fmt.Println(entrypoint())
}
