package main

import "encoding/json"

type P struct {
	Name string `json:"name"`
	Age  int    `json:"age,omitempty"`
}

func run() string {
	p := P{Name: "Cara"}
	b, _ := json.Marshal(p)
	return string(b)
}
