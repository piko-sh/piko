package main

func twice[T ordered](v T) T {
	return maxOf(v, v)
}
