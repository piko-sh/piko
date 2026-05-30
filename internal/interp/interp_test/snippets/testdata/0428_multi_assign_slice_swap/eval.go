package main

const gridSize = 4

func advance(current, next []int) {
	for row := 0; row < gridSize; row++ {
		rowUp := row - 1
		if rowUp < 0 {
			rowUp = gridSize - 1
		}
		rowDown := row + 1
		if rowDown >= gridSize {
			rowDown = 0
		}
		for col := 0; col < gridSize; col++ {
			colLeft := col - 1
			if colLeft < 0 {
				colLeft = gridSize - 1
			}
			colRight := col + 1
			if colRight >= gridSize {
				colRight = 0
			}
			n := current[rowUp*gridSize+colLeft] +
				current[rowUp*gridSize+col] +
				current[rowUp*gridSize+colRight] +
				current[row*gridSize+colLeft] +
				current[row*gridSize+colRight] +
				current[rowDown*gridSize+colLeft] +
				current[rowDown*gridSize+col] +
				current[rowDown*gridSize+colRight]
			alive := current[row*gridSize+col] == 1
			if alive {
				if n == 2 || n == 3 {
					next[row*gridSize+col] = 1
				} else {
					next[row*gridSize+col] = 0
				}
			} else {
				if n == 3 {
					next[row*gridSize+col] = 1
				} else {
					next[row*gridSize+col] = 0
				}
			}
		}
	}
}

func run() int {
	current := make([]int, gridSize*gridSize)
	next := make([]int, gridSize*gridSize)
	state := uint32(20260511)
	for cellIndex := 0; cellIndex < len(current); cellIndex++ {
		state = (state*1664525 + 1013904223) & 0xFFFFFFFF
		if (state>>30)&0x3 == 0 {
			current[cellIndex] = 1
		}
	}
	for g := 0; g < 5; g++ {
		advance(current, next)
		current, next = next, current
	}
	live := 0
	for _, v := range current {
		live += v
	}
	return live
}
