package main

func classify(s string) int {
	switch s {
	case "hello":
		return 1
	case "world":
		return 2
	case "foo":
		return 4
	default:
		return 0
	}
}

func run() int {
	return classify("hello") + classify("world") + classify("foo") + classify("bar")
}
