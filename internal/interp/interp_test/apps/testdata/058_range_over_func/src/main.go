package main

import "fmt"

func entrypoint() string {
	sum := 0
	for v := range evenInts(10) {
		sum += v
	}

	var labels []string
	for index, label := range pairs([]string{"a", "b", "c"}) {
		labels = append(labels, fmt.Sprintf("%d:%s", index, label))
	}

	earlyExit := 0
	for v := range evenInts(100) {
		if v >= 6 {
			break
		}
		earlyExit += v
	}

	return fmt.Sprintf("sum=%d labels=%v earlyExit=%d", sum, labels, earlyExit)
}

func main() {
	fmt.Println(entrypoint())
}
