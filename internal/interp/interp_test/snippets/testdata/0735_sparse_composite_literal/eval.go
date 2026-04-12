package main

import "fmt"

func run() string {
	result := ""

	arr := [10]int{0: 1, 5: 7, 9: 99}
	for _, v := range arr {
		result += fmt.Sprintf("%d,", v)
	}
	result += ";"

	mixed := [5]string{2: "two", 4: "four"}
	for _, v := range mixed {
		result += fmt.Sprintf("%q,", v)
	}
	result += ";"

	auto := [...]int{0: 10, 3: 40, 1: 20}
	result += fmt.Sprintf("auto=%v,len=%d", auto, len(auto))

	return result
}
