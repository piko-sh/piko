package main

import (
	"fmt"
	"testpkg/model"
)

func entrypoint() string {
	p := model.NewPerson("Alice", 30)
	return fmt.Sprintf("%s:%d", p.Name, p.Age)
}

func main() {
	fmt.Println(entrypoint())
}
