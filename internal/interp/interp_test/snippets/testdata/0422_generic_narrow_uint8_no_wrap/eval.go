package main

func wrapAdd[T ~uint8](a T, b T) T {
	return a + b
}

func run() int {
	r := wrapAdd[uint8](255, 1)
	return int(r)
}
