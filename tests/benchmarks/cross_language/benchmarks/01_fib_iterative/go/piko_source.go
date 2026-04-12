package main

import "time"

const fibTermsToCompute = 100000

const fibModulusBitmask = (1 << 64) - 1

func Run() string {
	value := computeFib(fibTermsToCompute)
	return uint64ToDecimalString(value)
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last uint64
	for index := 0; index < k; index++ {
		last = computeFib(fibTermsToCompute)
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return uint64ToDecimalString(last), elapsedNanos
}

func computeFib(n int) uint64 {
	previous := uint64(0)
	current := uint64(1)
	for index := 0; index < n; index++ {
		next := (previous + current) & fibModulusBitmask
		previous = current
		current = next
	}
	return current
}

func uint64ToDecimalString(value uint64) string {
	if value == 0 {
		return "0"
	}
	digits := [21]byte{}
	position := len(digits)
	for value > 0 {
		position--
		digits[position] = byte('0' + (value % 10))
		value /= 10
	}
	return string(digits[position:])
}
