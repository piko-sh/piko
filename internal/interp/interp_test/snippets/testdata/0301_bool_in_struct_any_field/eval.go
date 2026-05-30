package main

type Holder struct {
	Value any
}

func run() string {
	s := "hello"
	h := Holder{Value: s != ""}
	b, ok := h.Value.(bool)
	if !ok {
		return "type assertion failed"
	}
	if b {
		return "ok"
	}
	return "fail"
}
