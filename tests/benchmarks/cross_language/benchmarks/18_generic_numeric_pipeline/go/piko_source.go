package main

import "time"

const genericLcgMask = 0xFFFFFFFF

const slicesPerType = 500

const sliceSize = 100

const genericSeed = 0xCAFEBABE

type Numeric interface {
	uint32 | int64
}

func GenericSum[T Numeric](xs []T) T {
	var accumulator T
	for index := 0; index < len(xs); index++ {
		accumulator = accumulator + xs[index]
	}
	return accumulator
}

func GenericMax[T Numeric](xs []T) T {
	if len(xs) == 0 {
		var zero T
		return zero
	}
	maximum := xs[0]
	for index := 1; index < len(xs); index++ {
		if xs[index] > maximum {
			maximum = xs[index]
		}
	}
	return maximum
}

func GenericMin[T Numeric](xs []T) T {
	if len(xs) == 0 {
		var zero T
		return zero
	}
	minimum := xs[0]
	for index := 1; index < len(xs); index++ {
		if xs[index] < minimum {
			minimum = xs[index]
		}
	}
	return minimum
}

func Run() string {
	return doGenericPipeline()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doGenericPipeline()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doGenericPipeline() string {
	state := uint32(genericSeed)
	var accumulator uint32

	for sliceIndex := 0; sliceIndex < slicesPerType; sliceIndex++ {
		slice := make([]uint32, sliceSize)
		for elementIndex := 0; elementIndex < sliceSize; elementIndex++ {
			state = (state*1664525 + 1013904223) & genericLcgMask
			slice[elementIndex] = state
		}
		sumValue := GenericSum[uint32](slice)
		maxValue := GenericMax[uint32](slice)
		minValue := GenericMin[uint32](slice)
		accumulator = (accumulator ^ sumValue ^ maxValue ^ minValue) & genericLcgMask
	}

	for sliceIndex := 0; sliceIndex < slicesPerType; sliceIndex++ {
		slice := make([]int64, sliceSize)
		for elementIndex := 0; elementIndex < sliceSize; elementIndex++ {
			state = (state*1664525 + 1013904223) & genericLcgMask
			slice[elementIndex] = int64(state)
		}
		sumValue := GenericSum[int64](slice)
		maxValue := GenericMax[int64](slice)
		minValue := GenericMin[int64](slice)
		foldSum := uint32(sumValue & int64(genericLcgMask))
		foldMax := uint32(maxValue & int64(genericLcgMask))
		foldMin := uint32(minValue & int64(genericLcgMask))
		accumulator = (accumulator ^ foldSum ^ foldMax ^ foldMin) & genericLcgMask
	}

	return intToDecimal(int(accumulator))
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
