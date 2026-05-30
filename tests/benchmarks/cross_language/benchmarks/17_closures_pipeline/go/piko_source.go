package main

import "time"

const inputCount = 100000

const pipelineLcgMask = 0xFFFFFFFF

const pipelineSeed = 0xCAFEBABE

const foldRotateShift = 5

func makeMultiply(factor uint32) func(uint32) uint32 {
	return func(value uint32) uint32 {
		return (value * factor) & pipelineLcgMask
	}
}

func makeDivisorFilter(divisor uint32) func(uint32) bool {
	return func(value uint32) bool {
		return value%divisor != 0
	}
}

func makeAdd(offset uint32) func(uint32) uint32 {
	return func(value uint32) uint32 {
		return (value + offset) & pipelineLcgMask
	}
}

func makeThresholdFilter(threshold uint32) func(uint32) bool {
	return func(value uint32) bool {
		return value > threshold
	}
}

func makeFold(shift uint32) func(uint32, uint32) uint32 {
	return func(acc uint32, value uint32) uint32 {
		rotated := ((value << shift) | (value >> (32 - shift))) & pipelineLcgMask
		return (acc ^ rotated) & pipelineLcgMask
	}
}

func applyMap(input []uint32, fn func(uint32) uint32) []uint32 {
	output := make([]uint32, len(input))
	for index := 0; index < len(input); index++ {
		output[index] = fn(input[index])
	}
	return output
}

func applyFilter(input []uint32, predicate func(uint32) bool) []uint32 {
	output := make([]uint32, 0, len(input))
	for index := 0; index < len(input); index++ {
		current := input[index]
		if predicate(current) {
			output = append(output, current)
		}
	}
	return output
}

func applyReduce(input []uint32, initial uint32, fn func(uint32, uint32) uint32) uint32 {
	accumulator := initial
	for index := 0; index < len(input); index++ {
		accumulator = fn(accumulator, input[index])
	}
	return accumulator
}

func Run() string {
	return doClosuresPipeline()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doClosuresPipeline()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doClosuresPipeline() string {
	state := uint32(pipelineSeed)
	input := make([]uint32, inputCount)
	for index := 0; index < inputCount; index++ {
		state = (state*1664525 + 1013904223) & pipelineLcgMask
		input[index] = state
	}

	multiplier := makeMultiply(3)
	divisorFilter := makeDivisorFilter(7)
	adder := makeAdd(1234)
	thresholdFilter := makeThresholdFilter(1000)
	folder := makeFold(foldRotateShift)

	stage1 := applyMap(input, multiplier)
	stage2 := applyFilter(stage1, divisorFilter)
	stage3 := applyMap(stage2, adder)
	stage4 := applyFilter(stage3, thresholdFilter)
	result := applyReduce(stage4, 0, folder)
	return intToDecimal(int(result))
}

func intToDecimal(value int) string {
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
