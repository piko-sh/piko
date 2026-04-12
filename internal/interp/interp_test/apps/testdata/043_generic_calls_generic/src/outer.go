package main

func sevenTimes[T Number](v T) T {
	return tripled(v) + tripled(v) + v
}
