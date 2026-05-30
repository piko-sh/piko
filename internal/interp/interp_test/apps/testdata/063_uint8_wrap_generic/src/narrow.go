package main

func wrapAdd[T ~uint8 | ~uint16 | ~int8](a T, b T) T {
	return a + b
}

func wrapSub[T ~uint8 | ~uint16 | ~int8](a T, b T) T {
	return a - b
}
