package main

import "time"

const expressionCount = 10000

const resultMask = 0xFFFFFFFF

const lcgMask = 0xFFFFFFFF

func Run() string {
	return doExpressionEval()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doExpressionEval()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doExpressionEval() string {
	state := uint32(99887766)
	var runningSum uint32
	for expressionIndex := 0; expressionIndex < expressionCount; expressionIndex++ {
		var source string
		source, state = generateExpression(state)
		evaluator := evaluator{tokens: tokenise(source), position: 0}
		value := evaluator.parseExpression(0)
		runningSum = (runningSum + uint32(value)) & resultMask
	}
	return intToDecimalString(int(runningSum))
}

func generateExpression(state uint32) (string, uint32) {
	output := make([]byte, 0, 64)
	state = (state*1664525 + 1013904223) & lcgMask
	depthBudget := int(state&0x7) + 4
	state = generateTerm(&output, state, depthBudget)
	return string(output), state
}

func generateTerm(output *[]byte, state uint32, depthBudget int) uint32 {
	state = (state*1664525 + 1013904223) & lcgMask
	if depthBudget <= 0 || state&0x3 == 0 {
		value := int(state & 0xFF)
		*output = append(*output, intToDecimalBytes(value)...)
		return state
	}
	*output = append(*output, '(')
	state = generateTerm(output, state, depthBudget-1)
	state = (state*1664525 + 1013904223) & lcgMask
	operatorByte := byte('+')
	switch state & 0x3 {
	case 1:
		operatorByte = '-'
	case 2:
		operatorByte = '*'
	case 3:
		operatorByte = '+'
	}
	*output = append(*output, operatorByte)
	state = generateTerm(output, state, depthBudget-1)
	*output = append(*output, ')')
	return state
}

const (
	tokenInteger = 0

	tokenPlus = 1

	tokenMinus = 2

	tokenMultiply = 3

	tokenParenLeft = 4

	tokenParenRight = 5
)

type token struct {
	kind int

	value int
}

func tokenise(source string) []token {
	tokens := make([]token, 0, len(source))
	position := 0
	for position < len(source) {
		current := source[position]
		switch {
		case current >= '0' && current <= '9':
			value := 0
			for position < len(source) && source[position] >= '0' && source[position] <= '9' {
				value = value*10 + int(source[position]-'0')
				position++
			}
			tokens = append(tokens, token{kind: tokenInteger, value: value})
		case current == '+':
			tokens = append(tokens, token{kind: tokenPlus})
			position++
		case current == '-':
			tokens = append(tokens, token{kind: tokenMinus})
			position++
		case current == '*':
			tokens = append(tokens, token{kind: tokenMultiply})
			position++
		case current == '(':
			tokens = append(tokens, token{kind: tokenParenLeft})
			position++
		case current == ')':
			tokens = append(tokens, token{kind: tokenParenRight})
			position++
		default:
			position++
		}
	}
	return tokens
}

type evaluator struct {
	tokens []token

	position int
}

func (parser *evaluator) parseExpression(minimumPrecedence int) int {
	leftValue := parser.parseAtom()
	for parser.position < len(parser.tokens) {
		operatorToken := parser.tokens[parser.position]
		operatorPrecedence := precedenceFor(operatorToken.kind)
		if operatorPrecedence == 0 || operatorPrecedence < minimumPrecedence {
			break
		}
		parser.position++
		rightValue := parser.parseExpression(operatorPrecedence + 1)
		switch operatorToken.kind {
		case tokenPlus:
			leftValue = leftValue + rightValue
		case tokenMinus:
			leftValue = leftValue - rightValue
		case tokenMultiply:
			leftValue = leftValue * rightValue
		}
	}
	return leftValue
}

func (parser *evaluator) parseAtom() int {
	if parser.position >= len(parser.tokens) {
		return 0
	}
	current := parser.tokens[parser.position]
	if current.kind == tokenInteger {
		parser.position++
		return current.value
	}
	if current.kind == tokenParenLeft {
		parser.position++
		value := parser.parseExpression(0)
		if parser.position < len(parser.tokens) && parser.tokens[parser.position].kind == tokenParenRight {
			parser.position++
		}
		return value
	}
	return 0
}

func precedenceFor(kind int) int {
	switch kind {
	case tokenMultiply:
		return 20
	case tokenPlus, tokenMinus:
		return 10
	default:
		return 0
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
