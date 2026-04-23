package main

func run() string {
	s := "hello"
	var out []any
	out = append(out, s != "")
	b, ok := out[0].(bool)
	if !ok {
		return "type assertion failed"
	}
	if b {
		return "ok"
	}
	return "fail"
}
