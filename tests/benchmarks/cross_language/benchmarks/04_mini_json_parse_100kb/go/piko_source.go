package main

import "time"

const targetBytes = 100 * 1024

const lcgMask = 0xFFFFFFFF

func Run() string {
	return doParse()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doParse()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doParse() string {
	document := generateJSONDocument()
	parser := jsonParser{source: document, position: 0, keysObserved: 0}
	parser.parseValue()
	return intToDecimalString(parser.keysObserved)
}

func generateJSONDocument() []byte {
	output := make([]byte, 0, targetBytes+1024)
	output = append(output, '{')
	state := uint32(31415926)
	entryIndex := 0
	for len(output) < targetBytes {
		if entryIndex > 0 {
			output = append(output, ',')
		}
		state = writeJSONString(&output, state)
		output = append(output, ':')
		state = writeJSONValue(&output, state, 1)
		entryIndex++
	}
	output = append(output, '}')
	return output
}

func writeJSONValue(output *[]byte, state uint32, depth int) uint32 {
	state = (state*1664525 + 1013904223) & lcgMask
	if len(*output) >= targetBytes {
		*output = append(*output, '0')
		return state
	}
	pick := state & 0x7
	if depth > 6 {
		pick = pick & 0x3
	}
	switch pick {
	case 0, 1:
		return writeJSONObject(output, state, depth)
	case 2:
		return writeJSONArray(output, state, depth)
	case 3, 4:
		return writeJSONString(output, state)
	case 5, 6:
		return writeJSONNumber(output, state)
	default:
		state = (state*1664525 + 1013904223) & lcgMask
		if state&1 == 0 {
			*output = append(*output, 't', 'r', 'u', 'e')
		} else {
			*output = append(*output, 'n', 'u', 'l', 'l')
		}
		return state
	}
}

func writeJSONObject(output *[]byte, state uint32, depth int) uint32 {
	state = (state*1664525 + 1013904223) & lcgMask
	entryCount := int(state&0x7) + 1
	*output = append(*output, '{')
	for entryIndex := 0; entryIndex < entryCount; entryIndex++ {
		if entryIndex > 0 {
			*output = append(*output, ',')
		}
		state = writeJSONString(output, state)
		*output = append(*output, ':')
		state = writeJSONValue(output, state, depth+1)
		if len(*output) >= targetBytes {
			break
		}
	}
	*output = append(*output, '}')
	return state
}

func writeJSONArray(output *[]byte, state uint32, depth int) uint32 {
	state = (state*1664525 + 1013904223) & lcgMask
	entryCount := int(state&0x7) + 1
	*output = append(*output, '[')
	for entryIndex := 0; entryIndex < entryCount; entryIndex++ {
		if entryIndex > 0 {
			*output = append(*output, ',')
		}
		state = writeJSONValue(output, state, depth+1)
		if len(*output) >= targetBytes {
			break
		}
	}
	*output = append(*output, ']')
	return state
}

func writeJSONString(output *[]byte, state uint32) uint32 {
	state = (state*1664525 + 1013904223) & lcgMask
	length := int(state&0x7) + 1
	*output = append(*output, '"')
	for charIndex := 0; charIndex < length; charIndex++ {
		state = (state*1664525 + 1013904223) & lcgMask
		*output = append(*output, byte('a'+(state%26)))
	}
	*output = append(*output, '"')
	return state
}

func writeJSONNumber(output *[]byte, state uint32) uint32 {
	state = (state*1664525 + 1013904223) & lcgMask
	value := int(state & 0xFFFF)
	*output = append(*output, intToDecimalBytes(value)...)
	return state
}

type jsonParser struct {
	source []byte

	position int

	keysObserved int
}

func (parser *jsonParser) parseValue() {
	parser.skipWhitespace()
	if parser.position >= len(parser.source) {
		return
	}
	current := parser.source[parser.position]
	switch current {
	case '{':
		parser.parseObject()
	case '[':
		parser.parseArray()
	case '"':
		parser.parseString()
	case 't':
		parser.position += 4
	case 'f':
		parser.position += 5
	case 'n':
		parser.position += 4
	default:
		parser.parseNumber()
	}
}

func (parser *jsonParser) parseObject() {
	parser.position++
	parser.skipWhitespace()
	for parser.position < len(parser.source) && parser.source[parser.position] != '}' {
		parser.skipWhitespace()
		parser.parseString()
		parser.keysObserved++
		parser.skipWhitespace()
		parser.position++
		parser.parseValue()
		parser.skipWhitespace()
		if parser.position < len(parser.source) && parser.source[parser.position] == ',' {
			parser.position++
		}
	}
	if parser.position < len(parser.source) {
		parser.position++
	}
}

func (parser *jsonParser) parseArray() {
	parser.position++
	parser.skipWhitespace()
	for parser.position < len(parser.source) && parser.source[parser.position] != ']' {
		parser.parseValue()
		parser.skipWhitespace()
		if parser.position < len(parser.source) && parser.source[parser.position] == ',' {
			parser.position++
		}
	}
	if parser.position < len(parser.source) {
		parser.position++
	}
}

func (parser *jsonParser) parseString() {
	parser.position++
	for parser.position < len(parser.source) && parser.source[parser.position] != '"' {
		parser.position++
	}
	if parser.position < len(parser.source) {
		parser.position++
	}
}

func (parser *jsonParser) parseNumber() {
	for parser.position < len(parser.source) {
		current := parser.source[parser.position]
		if current < '0' || current > '9' {
			break
		}
		parser.position++
	}
}

func (parser *jsonParser) skipWhitespace() {
	for parser.position < len(parser.source) {
		current := parser.source[parser.position]
		if current != ' ' && current != '\t' && current != '\n' && current != '\r' {
			break
		}
		parser.position++
	}
}

func intToDecimalBytes(value int) []byte {
	if value == 0 {
		return []byte{'0'}
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
	return append([]byte{}, digits[position:]...)
}

func intToDecimalString(value int) string {
	return string(intToDecimalBytes(value))
}
