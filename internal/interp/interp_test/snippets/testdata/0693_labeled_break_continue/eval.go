package main

import "fmt"

func findInMatrix(m [][]int, target int) (int, int, bool) {
	foundRow := -1
	foundCol := -1
Search:
	for i, row := range m {
		for j, v := range row {
			if v == target {
				foundRow = i
				foundCol = j
				break Search
			}
		}
	}
	if foundRow >= 0 {
		return foundRow, foundCol, true
	}
	return -1, -1, false
}

func sumNonZeroRows(m [][]int) int {
	total := 0
Outer:
	for _, row := range m {
		for _, v := range row {
			if v == 0 {
				continue Outer
			}
			total += v
		}
	}
	return total
}

func switchInsideLoop(values []int) string {
	result := ""
Loop:
	for _, v := range values {
		switch {
		case v < 0:
			result += "neg-stop;"
			break Loop
		case v == 0:
			result += "zero-skip;"
			continue Loop
		default:
			result += fmt.Sprintf("%d;", v)
		}
	}
	return result
}

func run() string {
	matrix := [][]int{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}}
	row, col, ok := findInMatrix(matrix, 5)
	out := fmt.Sprintf("found:%d/%d/%v;", row, col, ok)

	scattered := [][]int{{1, 2, 3}, {4, 0, 6}, {7, 8, 9}}
	out += fmt.Sprintf("sum:%d;", sumNonZeroRows(scattered))

	out += "switch:" + switchInsideLoop([]int{1, 2, 0, 3, -1, 4})

	return out
}
