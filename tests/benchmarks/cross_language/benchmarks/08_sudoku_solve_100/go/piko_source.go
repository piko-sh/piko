package main

import "time"

const puzzleCount = 100

const cellsToRemove = 40

const resultMask = 0xFFFFFFFF

const lcgMask = 0xFFFFFFFF

var seedGrid = [9][9]int{
	{5, 3, 4, 6, 7, 8, 9, 1, 2},
	{6, 7, 2, 1, 9, 5, 3, 4, 8},
	{1, 9, 8, 3, 4, 2, 5, 6, 7},
	{8, 5, 9, 7, 6, 1, 4, 2, 3},
	{4, 2, 6, 8, 5, 3, 7, 9, 1},
	{7, 1, 3, 9, 2, 4, 8, 5, 6},
	{9, 6, 1, 5, 3, 7, 2, 8, 4},
	{2, 8, 7, 4, 1, 9, 6, 3, 5},
	{3, 4, 5, 2, 8, 6, 1, 7, 9},
}

func Run() string {
	return doSudoku()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doSudoku()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doSudoku() string {
	state := uint32(76543210)
	var foldHash uint32
	board := [81]int{}
	for puzzleIndex := 0; puzzleIndex < puzzleCount; puzzleIndex++ {
		state = generatePuzzle(&board, state)
		solveSudoku(&board, 0)
		for cellIndex := 0; cellIndex < 81; cellIndex++ {
			foldHash = ((foldHash * 31) + uint32(board[cellIndex])) & resultMask
		}
	}
	return intToDecimalString(int(foldHash))
}

func generatePuzzle(board *[81]int, state uint32) uint32 {
	for rowIndex := 0; rowIndex < 9; rowIndex++ {
		for columnIndex := 0; columnIndex < 9; columnIndex++ {
			board[rowIndex*9+columnIndex] = seedGrid[rowIndex][columnIndex]
		}
	}
	for removalIndex := 0; removalIndex < cellsToRemove; removalIndex++ {
		state = (state*1664525 + 1013904223) & lcgMask
		position := int(state % 81)
		board[position] = 0
	}
	return state
}

func solveSudoku(board *[81]int, position int) bool {
	for position < 81 && board[position] != 0 {
		position++
	}
	if position == 81 {
		return true
	}
	rowIndex := position / 9
	columnIndex := position % 9
	for candidate := 1; candidate <= 9; candidate++ {
		if isValidPlacement(board, rowIndex, columnIndex, candidate) {
			board[position] = candidate
			if solveSudoku(board, position+1) {
				return true
			}
		}
	}
	board[position] = 0
	return false
}

func isValidPlacement(board *[81]int, rowIndex, columnIndex, candidate int) bool {
	for index := 0; index < 9; index++ {
		if board[rowIndex*9+index] == candidate {
			return false
		}
		if board[index*9+columnIndex] == candidate {
			return false
		}
	}
	boxRow := (rowIndex / 3) * 3
	boxColumn := (columnIndex / 3) * 3
	for offsetRow := 0; offsetRow < 3; offsetRow++ {
		for offsetColumn := 0; offsetColumn < 3; offsetColumn++ {
			if board[(boxRow+offsetRow)*9+(boxColumn+offsetColumn)] == candidate {
				return false
			}
		}
	}
	return true
}

func intToDecimalString(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := [20]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		position--
		digits[position] = '-'
	}
	return string(digits[position:])
}
