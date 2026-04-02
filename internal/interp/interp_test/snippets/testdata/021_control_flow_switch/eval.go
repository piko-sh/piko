package main

func run() int {
	x := 3
	result := 0
	switch x {
	case 1, 2:
		result = 10
	case 3, 4:
		result = 20
	default:
		result = 30
	}
	return result
}
