package main

func describe[T any](v T) string {
	var x any = v
	switch x.(type) {
	case int:
		return "int"
	case string:
		return "string"
	case bool:
		return "bool"
	default:
		return "other"
	}
}

func run() string {
	a := describe(42)
	b := describe("hi")
	c := describe(true)
	d := describe(3.14)
	return a + "," + b + "," + c + "," + d
}
