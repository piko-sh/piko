package main

func run() int {
	var x any = 3.14
	switch v := x.(type) {
	case int, float64:
		_ = v
		return 1
	case string:
		return 2
	default:
		return 3
	}
}
