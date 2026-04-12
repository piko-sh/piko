package main

import "time"

const dictionaryWordCount = 50000

const queryCount = 50000

const minLength = 4
const maxLengthMinusOne = 19
const lengthRangeMask = 0xF

const alphabet = "abcdefghijklmnop"
const alphabetMask = 0xF

const lcgMask = 0xFFFFFFFF

const resultMask = 0xFFFFFFFF

func Run() string {
	return doTrie()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doTrie()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doTrie() string {
	root := newTrieNode()
	state := uint32(98765432)
	for wordIndex := 0; wordIndex < dictionaryWordCount; wordIndex++ {
		var word string
		word, state = generateWord(state)
		trieInsert(root, word)
	}
	queryState := uint32(13579)
	var runningSum uint32
	for queryIndex := 0; queryIndex < queryCount; queryIndex++ {
		var query string
		query, queryState = generateWord(queryState)
		runningSum = (runningSum + uint32(trieLongestPrefixLength(root, query))) & resultMask
	}
	return intToDecimalString(int(runningSum))
}

type trieNode struct {
	children [16]*trieNode

	terminal bool
}

func newTrieNode() *trieNode {
	return &trieNode{}
}

func generateWord(state uint32) (string, uint32) {
	state = (state*1664525 + 1013904223) & lcgMask
	length := minLength + int(state&lengthRangeMask)
	output := make([]byte, length)
	for charIndex := 0; charIndex < length; charIndex++ {
		state = (state*1664525 + 1013904223) & lcgMask
		output[charIndex] = alphabet[int(state)&alphabetMask]
	}
	return string(output), state
}

func trieInsert(root *trieNode, word string) {
	node := root
	for charIndex := 0; charIndex < len(word); charIndex++ {
		slot := int(word[charIndex] - alphabet[0])
		if node.children[slot] == nil {
			node.children[slot] = newTrieNode()
		}
		node = node.children[slot]
	}
	node.terminal = true
}

func trieLongestPrefixLength(root *trieNode, query string) int {
	node := root
	longestTerminal := 0
	for charIndex := 0; charIndex < len(query); charIndex++ {
		slot := int(query[charIndex] - alphabet[0])
		next := node.children[slot]
		if next == nil {
			break
		}
		node = next
		if node.terminal {
			longestTerminal = charIndex + 1
		}
	}
	return longestTerminal
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
