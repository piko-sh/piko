package main

func run() string {
	s := "hello"
	m := map[string]any{
		"nonEmpty": s != "",
		"empty":    s == "",
	}
	b0, ok0 := m["nonEmpty"].(bool)
	b1, ok1 := m["empty"].(bool)
	if !ok0 || !ok1 {
		return "type assertion failed"
	}
	if b0 && !b1 {
		return "ok"
	}
	return "fail"
}
