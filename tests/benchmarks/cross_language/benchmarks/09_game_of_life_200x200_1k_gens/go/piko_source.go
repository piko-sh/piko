package main

import "time"

const gridSize = 200

const generationCount = 1000

const lcgMask = 0xFFFFFFFF

func Run() string {
	return doLife()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doLife()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doLife() string {
	current := make([]int, gridSize*gridSize)
	next := make([]int, gridSize*gridSize)
	state := uint32(20260511)
	for cellIndex := 0; cellIndex < len(current); cellIndex++ {
		state = (state*1664525 + 1013904223) & lcgMask
		if (state>>30)&0x3 == 0 {
			current[cellIndex] = 1
		}
	}
	for generationIndex := 0; generationIndex < generationCount; generationIndex++ {
		advanceGeneration(current, next)
		current, next = next, current
	}
	liveCount := 0
	for _, cellValue := range current {
		liveCount += cellValue
	}
	return intToDecimalString(liveCount)
}

func advanceGeneration(current, next []int) {
	for rowIndex := 0; rowIndex < gridSize; rowIndex++ {
		rowUp := rowIndex - 1
		if rowUp < 0 {
			rowUp = gridSize - 1
		}
		rowDown := rowIndex + 1
		if rowDown >= gridSize {
			rowDown = 0
		}
		rowOffset := rowIndex * gridSize
		rowUpOffset := rowUp * gridSize
		rowDownOffset := rowDown * gridSize
		for columnIndex := 0; columnIndex < gridSize; columnIndex++ {
			columnLeft := columnIndex - 1
			if columnLeft < 0 {
				columnLeft = gridSize - 1
			}
			columnRight := columnIndex + 1
			if columnRight >= gridSize {
				columnRight = 0
			}
			neighbours := current[rowUpOffset+columnLeft] +
				current[rowUpOffset+columnIndex] +
				current[rowUpOffset+columnRight] +
				current[rowOffset+columnLeft] +
				current[rowOffset+columnRight] +
				current[rowDownOffset+columnLeft] +
				current[rowDownOffset+columnIndex] +
				current[rowDownOffset+columnRight]
			alive := current[rowOffset+columnIndex] == 1
			if alive {
				if neighbours == 2 || neighbours == 3 {
					next[rowOffset+columnIndex] = 1
				} else {
					next[rowOffset+columnIndex] = 0
				}
			} else {
				if neighbours == 3 {
					next[rowOffset+columnIndex] = 1
				} else {
					next[rowOffset+columnIndex] = 0
				}
			}
		}
	}
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
