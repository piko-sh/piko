package main

import "encoding/json"

func run() string {
	b, _ := json.Marshal([]int{1, 2, 3})
	return string(b)
}
