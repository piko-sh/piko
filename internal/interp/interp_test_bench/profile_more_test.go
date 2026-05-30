// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

//go:build bench

package interp_test_bench

import (
	"testing"
)

const (
	sudokuSource = `
	package main
	func isValid(board [9][9]int64, row, col int, num int64) bool {
		for x := 0; x < 9; x++ {
			if board[row][x] == num {
				return false
			}
			if board[x][col] == num {
				return false
			}
		}
		startRow := (row / 3) * 3
		startCol := (col / 3) * 3
		for i := 0; i < 3; i++ {
			for j := 0; j < 3; j++ {
				if board[startRow+i][startCol+j] == num {
					return false
				}
			}
		}
		return true
	}
	func solve(board [9][9]int64) bool {
		for row := 0; row < 9; row++ {
			for col := 0; col < 9; col++ {
				if board[row][col] == 0 {
					for num := int64(1); num <= 9; num++ {
						if isValid(board, row, col, num) {
							board[row][col] = num
							if solve(board) {
								return true
							}
							board[row][col] = 0
						}
					}
					return false
				}
			}
		}
		return true
	}
	func EntrypointRun() int64 {
		var board [9][9]int64
		board[0][0] = 5
		board[0][1] = 3
		board[1][0] = 6
		board[1][3] = 1
		board[1][4] = 9
		board[1][5] = 5
		board[2][1] = 9
		board[2][2] = 8
		board[4][0] = 8
		board[4][4] = 6
		for i := 0; i < 5; i++ {
			_ = solve(board)
		}
		return board[0][0]
	}
	`
	brainfuckSource = `
	package main
	const program = "+++++[->+++++<]>[->+++++++<]>."
	func EntrypointRun() int64 {
		memory := make([]int64, 30000)
		var pointer int
		var programCounter int
		bracketStack := make([]int, 0, 64)
		for i := 0; i < 2000; i++ {
			pointer = 0
			programCounter = 0
			memory[0] = 0
			memory[1] = 0
			memory[2] = 0
			memory[3] = 0
			bracketStack = bracketStack[:0]
			for programCounter < len(program) {
				cmd := program[programCounter]
				if cmd == '+' {
					memory[pointer]++
				} else if cmd == '-' {
					memory[pointer]--
				} else if cmd == '>' {
					pointer++
				} else if cmd == '<' {
					pointer--
				} else if cmd == '[' {
					if memory[pointer] == 0 {
						depth := 1
						for depth > 0 {
							programCounter++
							if program[programCounter] == '[' {
								depth++
							} else if program[programCounter] == ']' {
								depth--
							}
						}
					} else {
						bracketStack = append(bracketStack, programCounter)
					}
				} else if cmd == ']' {
					if memory[pointer] != 0 {
						programCounter = bracketStack[len(bracketStack)-1]
					} else {
						bracketStack = bracketStack[:len(bracketStack)-1]
					}
				}
				programCounter++
			}
		}
		return memory[0]
	}
	`
	markovSource = `
	package main
	func EntrypointRun() int64 {
		words := []string{"hello", "world", "foo", "bar", "baz", "qux"}
		bigrams := map[string][]string{}
		for index := 0; index < 100; index++ {
			previous := words[index%len(words)]
			current := words[(index+1)%len(words)]
			bigrams[previous] = append(bigrams[previous], current)
		}
		var totalLength int64
		for index := 0; index < 1000; index++ {
			word := words[index%len(words)]
			nexts := bigrams[word]
			if len(nexts) > 0 {
				next := nexts[index%len(nexts)]
				totalLength += int64(len(next))
			}
		}
		return totalLength
	}
	`
	dijkstraSource = `
	package main
	type edge struct {
		target int
		weight int64
	}
	func EntrypointRun() int64 {
		const numNodes = 200
		graph := make([][]edge, numNodes)
		for index := 0; index < numNodes; index++ {
			graph[index] = []edge{}
		}
		for source := 0; source < numNodes; source++ {
			for offset := 1; offset <= 3; offset++ {
				target := (source + offset) % numNodes
				graph[source] = append(graph[source], edge{target: target, weight: int64(source + offset)})
			}
		}
		distance := make([]int64, numNodes)
		visited := make([]bool, numNodes)
		for index := 0; index < numNodes; index++ {
			distance[index] = 1 << 60
		}
		distance[0] = 0
		for iteration := 0; iteration < numNodes; iteration++ {
			var u int = -1
			var bestDistance int64 = 1 << 62
			for node := 0; node < numNodes; node++ {
				if !visited[node] && distance[node] < bestDistance {
					bestDistance = distance[node]
					u = node
				}
			}
			if u == -1 {
				break
			}
			visited[u] = true
			for index := 0; index < len(graph[u]); index++ {
				neighbour := graph[u][index]
				candidate := distance[u] + neighbour.weight
				if candidate < distance[neighbour.target] {
					distance[neighbour.target] = candidate
				}
			}
		}
		var sum int64
		for index := 0; index < numNodes; index++ {
			sum += distance[index]
		}
		return sum
	}
	`
)

func TestProfileSudoku(t *testing.T) {
	runProfile(t, "sudoku", sudokuSource, "/tmp/piko_sudoku.prof", 3)
}

func TestProfileBrainfuck(t *testing.T) {
	runProfile(t, "brainfuck", brainfuckSource, "/tmp/piko_bf.prof", 3)
}

func TestProfileMarkov(t *testing.T) {
	runProfile(t, "markov", markovSource, "/tmp/piko_markov.prof", 50)
}

func TestProfileDijkstra(t *testing.T) {
	runProfile(t, "dijkstra", dijkstraSource, "/tmp/piko_dijkstra.prof", 5)
}
