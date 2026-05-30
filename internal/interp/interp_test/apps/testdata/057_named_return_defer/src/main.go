package main

import "fmt"

func entrypoint() string {
	good, errGood := computeWithAdjustment(5)
	bad, errBad := computeWithAdjustment(-1)
	return fmt.Sprintf("good=%d errGood=%v bad=%d errBad=%v", good, errGood, bad, errBad)
}

func main() {
	fmt.Println(entrypoint())
}
