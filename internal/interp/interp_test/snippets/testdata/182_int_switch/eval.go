package main

func classify(x int) int {
	switch x {
	case 1:
		return 10
	case 2:
		return 20
	case 3:
		return 30
	default:
		return 0
	}
}

func run() int {
	return classify(1) + classify(2) + classify(3) + classify(4)
}
