package main

func run() string {
	apply := func(fn func(string) []any, input string) []any {
		return fn(input)
	}
	produce := func(s string) []any {
		return []any{s != "", s == "hello", s}
	}
	parts := apply(produce, "hello")
	b0 := parts[0].(bool)
	b1 := parts[1].(bool)
	if b0 && b1 {
		return "ok"
	}
	return "fail"
}
