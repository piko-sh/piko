package main

var x interface{} = 3.14

func run() string {
	result := "unknown"
	switch x.(type) {
	case int:
		result = "int"
	case string:
		result = "string"
	default:
		result = "other"
	}
	return result
}
