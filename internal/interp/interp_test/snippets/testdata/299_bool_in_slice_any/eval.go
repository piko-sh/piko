package main

func run() string {
	s := "hello"
	xs := []any{s != "", s == ""}
	b0, ok0 := xs[0].(bool)
	b1, ok1 := xs[1].(bool)
	if !ok0 || !ok1 {
		return "type assertion failed"
	}
	if b0 && !b1 {
		return "ok"
	}
	return "fail"
}
