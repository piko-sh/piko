package main

func evaluateOr(left, right any) any {
	if truthy(left) {
		return left
	}
	return right
}

func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	}
	return false
}

func run() string {
	s := "hello"
	empty := ""
	if evaluateOr(s != "", empty != "").(bool) {
		return "ok"
	}
	return "fail"
}
