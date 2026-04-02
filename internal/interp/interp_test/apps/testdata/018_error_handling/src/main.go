package main

import "fmt"

func entrypoint() int {
	v1, err1 := safeDivide(10, 2)
	_, err2 := safeDivide(10, 0)

	result := v1
	if err1 == nil {
		result += 5
	}
	if err2 != nil {
		result += 5
	}
	return result
}

func main() {
	fmt.Println(entrypoint())
}
