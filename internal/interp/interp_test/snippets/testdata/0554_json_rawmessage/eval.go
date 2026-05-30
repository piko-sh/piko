package main

import "encoding/json"

type Wrapper struct {
	Payload json.RawMessage
}

func run() string {
	w := Wrapper{Payload: json.RawMessage(`{"x":1}`)}
	b, _ := json.Marshal(w)
	return string(b)
}
