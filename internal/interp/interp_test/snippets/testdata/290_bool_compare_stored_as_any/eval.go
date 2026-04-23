package main

func run() string {
	s := "hello"
	empty := ""
	var boxed any = s != ""
	var boxedEmpty any = empty != ""
	b1, ok1 := boxed.(bool)
	b2, ok2 := boxedEmpty.(bool)
	if !ok1 || !ok2 {
		return "type assertion failed"
	}
	if b1 && !b2 {
		return "ok"
	}
	return "fail"
}
