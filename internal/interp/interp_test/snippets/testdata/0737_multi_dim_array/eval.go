package main

import "fmt"

func run() string {
	result := ""

	var grid [3][4]int
	for i := 0; i < 3; i++ {
		for j := 0; j < 4; j++ {
			grid[i][j] = i*4 + j
		}
	}
	for _, row := range grid {
		for _, v := range row {
			result += fmt.Sprintf("%d,", v)
		}
		result += "/"
	}
	result += ";"

	init2 := [2][3]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
	}
	for _, row := range init2 {
		for _, v := range row {
			result += v
		}
	}
	result += ";"

	var cube [2][2][2]int
	cube[1][1][1] = 7
	cube[0][1][0] = 3
	result += fmt.Sprintf("cube111=%d,cube010=%d", cube[1][1][1], cube[0][1][0])

	return result
}
