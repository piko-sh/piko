package main

import "time"

const typeSwitchMask = 0xFFFFFFFF

const valueStreamLength = 100000

const typeSwitchSeed = 0xC0FFEE

const fnvOffsetBasis32 = 0x811C9DC5
const fnvPrime32 = 0x01000193

type IntVal struct {
	v uint32
}

type StringVal struct {
	v string
}

type BytesVal struct {
	v []byte
}

type BoolVal struct {
	v bool
}

type FloatVal struct {
	v float64
}

func fnv1a32(key string) uint32 {
	hash := uint32(fnvOffsetBasis32)
	for index := 0; index < len(key); index++ {
		hash ^= uint32(key[index])
		hash = (hash * fnvPrime32) & typeSwitchMask
	}
	return hash
}

func dispatchByType(v any) uint32 {
	switch concrete := v.(type) {
	case *IntVal:
		return (concrete.v + 1234) & typeSwitchMask
	case *StringVal:
		return fnv1a32(concrete.v)
	case *BytesVal:
		var sum uint32
		for index := 0; index < len(concrete.v); index++ {
			sum = (sum + uint32(concrete.v[index])) & typeSwitchMask
		}
		return sum
	case *BoolVal:
		if concrete.v {
			return 1
		}
		return 0
	case *FloatVal:
		return uint32(concrete.v) & typeSwitchMask
	}
	return 0
}

func buildValueStream() []any {
	state := uint32(typeSwitchSeed)
	values := make([]any, valueStreamLength)
	sharedString := "abcdefghij"
	sharedBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	for index := 0; index < valueStreamLength; index++ {
		state = (state*1664525 + 1013904223) & typeSwitchMask
		kind := state & 0x7
		switch kind {
		case 0, 1, 2:
			values[index] = &IntVal{v: state}
		case 3:
			values[index] = &StringVal{v: sharedString}
		case 4:
			values[index] = &BytesVal{v: sharedBytes}
		case 5:
			values[index] = &BoolVal{v: (state & 1) == 1}
		case 6:
			values[index] = &FloatVal{v: float64(state)}
		default:
			values[index] = &IntVal{v: state}
		}
	}
	return values
}

func Run() string {
	return doTypeSwitches()
}

func RunInner(k int) (string, int64) {
	values := buildValueStream()
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = walkAndDispatch(values)
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doTypeSwitches() string {
	values := buildValueStream()
	return walkAndDispatch(values)
}

func walkAndDispatch(values []any) string {
	var accumulator uint32
	for index := 0; index < len(values); index++ {
		accumulator = (accumulator ^ dispatchByType(values[index])) & typeSwitchMask
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
