package main

import "encoding/json"

func run() string {
	s := "hello"
	payload := map[string]any{
		"nonEmpty": s != "",
		"empty":    s == "",
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "marshal error"
	}
	return string(raw)
}
