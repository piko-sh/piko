package main

type wordCount struct {
	word  string
	count int
}

var vocab = []string{"the", "of", "and", "to", "in", "a", "is", "it"}

func lessForTop(left, right wordCount) bool {
	if left.count != right.count {
		return left.count > right.count
	}
	return left.word < right.word
}

func run() int {
	output := make([]byte, 0, 1024*1024+16)
	state := uint32(987654321)
	for len(output) < 1024*1024 {
		state = state*1664525 + 1013904223
		word := vocab[int(state)&0x7]
		if len(output) > 0 {
			output = append(output, ' ')
		}
		for index := 0; index < len(word); index++ {
			output = append(output, word[index])
		}
	}

	counts := map[string]int{}
	tokenStart := 0
	for position := 0; position <= len(output); position++ {
		if position == len(output) || output[position] == ' ' {
			if position > tokenStart {
				word := string(output[tokenStart:position])
				counts[word] = counts[word] + 1
			}
			tokenStart = position + 1
		}
	}

	heap := make([]wordCount, 0, 4)
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
		if len(heap) > 3 {
			heap = heap[:3]
		}
	}
	total := 0
	for index := 0; index < len(heap); index++ {
		total += heap[index].count
	}
	return total
}
