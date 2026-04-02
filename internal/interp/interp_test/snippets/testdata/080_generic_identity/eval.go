package main

func identity[T any](v T) T {
	return v
}

func run() int {
	return identity(42)
}
