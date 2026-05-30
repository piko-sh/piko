package main

import "encoding/json"

func run() int {
	var x int
	err := json.Unmarshal([]byte(`{not json`), &x)
	if err != nil {
		return 1
	}
	return 0
}
