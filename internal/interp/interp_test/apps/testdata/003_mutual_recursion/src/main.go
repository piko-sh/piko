package main

import "fmt"

func isOdd(n int) bool {
	if n == 0 {
		return false
	}
	return isEven(n - 1)
}

func entrypoint() string {
	result := ""
	for i := 0; i < 6; i++ {
		if isEven(i) {
			result += "E"
		} else {
			result += "O"
		}
	}
	return result
}

func main() {
	fmt.Println(entrypoint())
}
