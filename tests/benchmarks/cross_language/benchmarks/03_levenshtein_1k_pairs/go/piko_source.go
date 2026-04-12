package main

import "time"

const pairCount = 1000

const alphabet = "abcdefghijklmnopqrstuvwxyz012345"
const alphabetMask = 0x1F

const minLength = 8
const maxLengthMinusOne = 39
const lengthRangeMask = 0x1F

const lcgMask = 0xFFFFFFFF

func Run() string {
	return doLevenshtein()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doLevenshtein()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doLevenshtein() string {
	lcgState := uint32(2718281828)
	var totalDistance int
	dpRow := make([]int, maxLengthMinusOne+2)
	dpNext := make([]int, maxLengthMinusOne+2)
	for pairIndex := 0; pairIndex < pairCount; pairIndex++ {
		var leftString, rightString string
		leftString, lcgState = generateString(lcgState)
		rightString, lcgState = generateString(lcgState)
		totalDistance += editDistance(leftString, rightString, dpRow, dpNext)
	}
	return intToDecimalString(totalDistance)
}

func generateString(state uint32) (string, uint32) {
	state = (state*1664525 + 1013904223) & lcgMask
	length := minLength + int(state&lengthRangeMask)
	output := make([]byte, length)
	for position := 0; position < length; position++ {
		state = (state*1664525 + 1013904223) & lcgMask
		output[position] = alphabet[int(state)&alphabetMask]
	}
	return string(output), state
}

func editDistance(left, right string, dpRow, dpNext []int) int {
	leftLength := len(left)
	rightLength := len(right)
	for columnIndex := 0; columnIndex <= rightLength; columnIndex++ {
		dpRow[columnIndex] = columnIndex
	}
	for rowIndex := 1; rowIndex <= leftLength; rowIndex++ {
		dpNext[0] = rowIndex
		leftCharacter := left[rowIndex-1]
		for columnIndex := 1; columnIndex <= rightLength; columnIndex++ {
			substitutionCost := 1
			if leftCharacter == right[columnIndex-1] {
				substitutionCost = 0
			}
			deletion := dpRow[columnIndex] + 1
			insertion := dpNext[columnIndex-1] + 1
			substitution := dpRow[columnIndex-1] + substitutionCost
			candidate := deletion
			if insertion < candidate {
				candidate = insertion
			}
			if substitution < candidate {
				candidate = substitution
			}
			dpNext[columnIndex] = candidate
		}
		for swapIndex := 0; swapIndex <= rightLength; swapIndex++ {
			dpRow[swapIndex] = dpNext[swapIndex]
		}
	}
	return dpRow[rightLength]
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
