package main

import "encoding/json"

type P struct {
	Name string
	Age  int
}

func run() int {
	var p P
	_ = json.Unmarshal([]byte(`{"Name":"Bob","Age":25}`), &p)
	if p.Name == "Bob" && p.Age == 25 {
		return 1
	}
	return 0
}
