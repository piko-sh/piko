package main

import "time"

const corpusBytes = 1024 * 1024

const topK = 10

var vocabulary = []string{
	"the", "of", "and", "to", "in", "a", "is", "it", "you", "that",
	"he", "was", "for", "on", "are", "with", "as", "I", "his", "they",
	"be", "at", "one", "have", "this", "from", "or", "had", "by", "hot",
	"word", "but", "what", "some", "we", "can", "out", "other", "were", "all",
	"there", "when", "up", "use", "your", "how", "said", "an", "each", "she",
	"which", "do", "their", "time", "if", "will", "way", "about", "many", "then",
	"them", "write", "would", "like",
}

const vocabularyMask = 0x3F

func Run() string {
	return doWordFrequency()
}

func RunInner(k int) (string, int64) {
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doWordFrequency()
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func doWordFrequency() string {
	corpus := string(generateCorpus())
	counts := countWordsManual(corpus)
	top := topKByInsertionSort(counts, topK)
	return formatTopKOutput(top)
}

func generateCorpus() []byte {
	output := make([]byte, 0, corpusBytes+16)
	lcgState := uint32(987654321)
	for len(output) < corpusBytes {
		lcgState = lcgState*1664525 + 1013904223
		word := vocabulary[int(lcgState)&vocabularyMask]
		if len(output) > 0 {
			output = append(output, ' ')
		}
		for stringIndex := 0; stringIndex < len(word); stringIndex++ {
			output = append(output, word[stringIndex])
		}
	}
	return output
}

func countWordsManual(corpus string) map[string]int {
	counts := map[string]int{}
	tokenStart := 0
	corpusLength := len(corpus)
	for position := 0; position <= corpusLength; position++ {
		if position == corpusLength || corpus[position] == ' ' {
			if position > tokenStart {
				word := corpus[tokenStart:position]
				counts[word] = counts[word] + 1
			}
			tokenStart = position + 1
		}
	}
	return counts
}

type wordCount struct {
	word string

	count int
}

func topKByInsertionSort(counts map[string]int, k int) []wordCount {
	heap := make([]wordCount, 0, k+1)
	for word, count := range counts {
		entry := wordCount{word: word, count: count}
		insertPosition := len(heap)
		for insertPosition > 0 && lessForTop(entry, heap[insertPosition-1]) {
			insertPosition--
		}
		heap = append(heap, wordCount{})
		for shiftPosition := len(heap) - 1; shiftPosition > insertPosition; shiftPosition-- {
			heap[shiftPosition] = heap[shiftPosition-1]
		}
		heap[insertPosition] = entry
		if len(heap) > k {
			heap = heap[:k]
		}
	}
	return heap
}

func lessForTop(left, right wordCount) bool {
	if left.count != right.count {
		return left.count > right.count
	}
	return left.word < right.word
}

func formatTopKOutput(top []wordCount) string {
	pieces := make([]byte, 0, 256)
	for lineIndex, entry := range top {
		if lineIndex > 0 {
			pieces = append(pieces, '\n')
		}
		for stringIndex := 0; stringIndex < len(entry.word); stringIndex++ {
			pieces = append(pieces, entry.word[stringIndex])
		}
		pieces = append(pieces, '\t')
		pieces = append(pieces, intToDecimalBytes(entry.count)...)
	}
	return string(pieces)
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
