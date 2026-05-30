package main

import "encoding/json"

func run() string {
	b, _ := json.MarshalIndent([]int{1, 2}, "", "  ")
	return string(b)
}
