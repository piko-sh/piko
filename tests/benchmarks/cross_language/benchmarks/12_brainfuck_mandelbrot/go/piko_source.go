package main

import "time"

const brainfuckProgram = "" +

	"++++++++++++>++++++++>" +

	"<<[" +

	">[->+>+<<]" +
	">>[-<<+>>]" +

	"<<<-]" +

	"++++++++[>++++++++<-]>." +

	"<++++++++++[->+++<]>++++."

const memorySize = 30000

const resultMask = 0xFFFFFFFF

func Run() string {
	return doBrainfuck()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doBrainfuck()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doBrainfuck() string {
	memory := make([]byte, memorySize)
	pointer := 0
	jumpTable := buildJumpTable(brainfuckProgram)
	instructionIndex := 0
	var foldHash uint32
	for instructionIndex < len(brainfuckProgram) {
		instruction := brainfuckProgram[instructionIndex]
		switch instruction {
		case '>':
			pointer++
		case '<':
			pointer--
		case '+':
			memory[pointer]++
		case '-':
			memory[pointer]--
		case '.':
			foldHash = ((foldHash * 31) + uint32(memory[pointer])) & resultMask
		case ',':
			memory[pointer] = 0
		case '[':
			if memory[pointer] == 0 {
				instructionIndex = jumpTable[instructionIndex]
			}
		case ']':
			if memory[pointer] != 0 {
				instructionIndex = jumpTable[instructionIndex]
			}
		}
		instructionIndex++
	}
	return intToDecimalString(int(foldHash))
}

func buildJumpTable(source string) []int {
	table := make([]int, len(source))
	stack := make([]int, 0, 32)
	for index := 0; index < len(source); index++ {
		character := source[index]
		if character == '[' {
			stack = append(stack, index)
			continue
		}
		if character == ']' {
			openIndex := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			table[openIndex] = index
			table[index] = openIndex
		}
	}
	return table
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
