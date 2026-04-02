package main

func swap[A, B any](a A, b B) (B, A) {
	return b, a
}

func run() string {
	b, _ := swap(42, "hello")
	return b
}
