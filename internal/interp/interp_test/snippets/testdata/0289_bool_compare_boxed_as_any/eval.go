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
	case string:
		return x != ""
	}
	return false
}

func run() string {
	a := "hello"
	b := ""
	leftNonEmpty := evaluateOr(a != "", b != "").(bool)
	rightNonEmpty := evaluateOr(b != "", a != "").(bool)
	bothEmpty := evaluateOr(b != "", b != "").(bool)
	if leftNonEmpty && rightNonEmpty && !bothEmpty {
		return "ok"
	}
	return "fail"
}
