package main

import "time"

const trainingTokens = 50000

const outputTokens = 10000

const resultMask = 0xFFFFFFFF

const lcgMask = 0xFFFFFFFF

const vocabularyMask = 0x3F

var vocabulary = []string{
	"the", "of", "and", "to", "in", "a", "is", "it", "you", "that",
	"he", "was", "for", "on", "are", "with", "as", "I", "his", "they",
	"be", "at", "one", "have", "this", "from", "or", "had", "by", "hot",
	"word", "but", "what", "some", "we", "can", "out", "other", "were", "all",
	"there", "when", "up", "use", "your", "how", "said", "an", "each", "she",
	"which", "do", "their", "time", "if", "will", "way", "about", "many", "then",
	"them", "write", "would", "like",
}

func Run() string {
	return doMarkov()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doMarkov()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doMarkov() string {
	corpus := generateCorpus(trainingTokens)
	transitions := buildBigramTable(corpus)
	state := uint32(42100100)
	var foldHash uint32
	previous := corpus[0]
	for outputIndex := 0; outputIndex < outputTokens; outputIndex++ {
		nextCandidates, present := transitions[previous]
		var nextToken string
		if !present || len(nextCandidates) == 0 {
			nextToken = corpus[0]
		} else {
			state = (state*1664525 + 1013904223) & lcgMask
			nextToken = nextCandidates[int(state)%len(nextCandidates)]
		}
		for stringIndex := 0; stringIndex < len(nextToken); stringIndex++ {
			foldHash = ((foldHash * 31) + uint32(nextToken[stringIndex])) & resultMask
		}
		previous = nextToken
	}
	return intToDecimalString(int(foldHash))
}

func generateCorpus(tokenCount int) []string {
	state := uint32(31415927)
	output := make([]string, tokenCount)
	for tokenIndex := 0; tokenIndex < tokenCount; tokenIndex++ {
		state = (state*1664525 + 1013904223) & lcgMask
		output[tokenIndex] = vocabulary[int(state)&vocabularyMask]
	}
	return output
}

func buildBigramTable(corpus []string) map[string][]string {
	transitions := map[string][]string{}
	for tokenIndex := 0; tokenIndex < len(corpus)-1; tokenIndex++ {
		previous := corpus[tokenIndex]
		next := corpus[tokenIndex+1]
		transitions[previous] = append(transitions[previous], next)
	}
	return transitions
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
