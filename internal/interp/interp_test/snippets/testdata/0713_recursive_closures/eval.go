package main

import "fmt"

func run() string {
	result := ""

	var fact func(int) int
	fact = func(n int) int {
		if n <= 1 {
			return 1
		}
		return n * fact(n-1)
	}
	result += fmt.Sprintf("fact5:%d;", fact(5))

	var isEven func(int) bool
	var isOdd func(int) bool
	isEven = func(n int) bool {
		if n == 0 {
			return true
		}
		return isOdd(n - 1)
	}
	isOdd = func(n int) bool {
		if n == 0 {
			return false
		}
		return isEven(n - 1)
	}
	result += fmt.Sprintf("even4:%v;odd5:%v;", isEven(4), isOdd(5))

	state := 0
	var add func(int)
	add = func(n int) {
		if n <= 0 {
			return
		}
		state += n
		add(n - 1)
	}
	add(5)
	result += fmt.Sprintf("captured:%d;", state)

	multiplier := 10
	var fib func(int) int
	fib = func(n int) int {
		if n < 2 {
			return n * multiplier
		}
		return fib(n-1) + fib(n-2)
	}
	result += fmt.Sprintf("fib6:%d", fib(6))

	return result
}
