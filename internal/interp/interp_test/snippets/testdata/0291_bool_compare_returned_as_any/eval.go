package main

func firstNonEmptyAsAny(a, b string) any {
	return a != "" || b != ""
}

func run() string {
	result := firstNonEmptyAsAny("x", "").(bool)
	if result {
		return "ok"
	}
	return "fail"
}
