package main

import "encoding/json"

type P struct {
	Name string
	Age  int
}

func run() string {
	p := P{Name: "Alice", Age: 30}
	b, _ := json.Marshal(p)
	return string(b)
}
