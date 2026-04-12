package main

import "fmt"

func count(yield func(int) bool) {
	for i := 1; i <= 5; i++ {
		if !yield(i) {
			return
		}
	}
}

func pairs(yield func(string, int) bool) {
	keys := []string{"a", "b", "c"}
	values := []int{10, 20, 30}
	for index := range keys {
		if !yield(keys[index], values[index]) {
			return
		}
	}
}

func sumSingleValue() int {
	total := 0
	for value := range count {
		total += value
	}
	return total
}

func concatPairs() string {
	result := ""
	for key, value := range pairs {
		result += fmt.Sprintf("%s=%d,", key, value)
	}
	return result
}

func sumWithBreak() int {
	total := 0
	for value := range count {
		if value > 3 {
			break
		}
		total += value
	}
	return total
}

func run() string {
	return fmt.Sprintf("sum=%d;pairs=%s;sumBreak=%d", sumSingleValue(), concatPairs(), sumWithBreak())
}
