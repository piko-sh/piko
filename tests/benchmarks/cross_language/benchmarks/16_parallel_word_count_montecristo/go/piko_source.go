package main

import (
	"os"
	"sync"
	"time"
)

const numberOfWorkers = 16

const topResultsCount = 50

func Run() string {
	corpus := loadCorpus()
	return doParallelWordCount(corpus)
}

func RunInner(k int) (string, int64) {
	corpus := loadCorpus()
	startNanos := time.Now().UnixNano()
	var last string
	for index := 0; index < k; index++ {
		last = doParallelWordCount(corpus)
	}
	elapsedNanos := time.Now().UnixNano() - startNanos
	return last, elapsedNanos
}

func loadCorpus() []byte {
	candidates := []string{
		"testdata/corpus.txt",
		"benchmarks/16_parallel_word_count_montecristo/testdata/corpus.txt",
		"tests/benchmarks/cross_language/benchmarks/16_parallel_word_count_montecristo/testdata/corpus.txt",
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return data
		}
	}
	panic("corpus not found at any candidate path")
}

func doParallelWordCount(corpus []byte) string {
	chunks := splitCorpusIntoChunks(corpus, numberOfWorkers)
	results := make([]map[string]int, numberOfWorkers)
	var waitGroup sync.WaitGroup
	for workerIndex := 0; workerIndex < numberOfWorkers; workerIndex++ {
		waitGroup.Add(1)
		go func(index int, chunk []byte) {
			defer waitGroup.Done()
			results[index] = tokeniseAndCount(chunk)
		}(workerIndex, chunks[workerIndex])
	}
	waitGroup.Wait()
	merged := mergeMaps(results)
	return formatTopK(merged, topResultsCount)
}

func splitCorpusIntoChunks(corpus []byte, chunkCount int) [][]byte {
	if len(corpus) == 0 || chunkCount <= 1 {
		return [][]byte{corpus}
	}
	chunkSize := len(corpus) / chunkCount
	chunks := make([][]byte, 0, chunkCount)
	cursor := 0
	for index := 0; index < chunkCount-1; index++ {
		boundary := cursor + chunkSize
		if boundary > len(corpus) {
			boundary = len(corpus)
		}
		for boundary < len(corpus) && isLetterByte(corpus[boundary]) {
			boundary++
		}
		chunks = append(chunks, corpus[cursor:boundary])
		cursor = boundary
	}
	chunks = append(chunks, corpus[cursor:])
	return chunks
}

func tokeniseAndCount(chunk []byte) map[string]int {
	counts := make(map[string]int)
	tokenStart := -1
	for index := 0; index < len(chunk); index++ {
		current := chunk[index]
		if isLetterByte(current) {
			if tokenStart < 0 {
				tokenStart = index
			}
		} else if tokenStart >= 0 {
			emitWord(counts, chunk[tokenStart:index])
			tokenStart = -1
		}
	}
	if tokenStart >= 0 {
		emitWord(counts, chunk[tokenStart:])
	}
	return counts
}

func emitWord(counts map[string]int, word []byte) {
	lowered := make([]byte, len(word))
	for index := 0; index < len(word); index++ {
		current := word[index]
		if current >= 'A' && current <= 'Z' {
			current += 32
		}
		lowered[index] = current
	}
	counts[string(lowered)]++
}

func isLetterByte(value byte) bool {
	if value >= 'a' && value <= 'z' {
		return true
	}
	if value >= 'A' && value <= 'Z' {
		return true
	}
	return false
}

func mergeMaps(maps []map[string]int) map[string]int {
	merged := make(map[string]int)
	for _, local := range maps {
		for key, value := range local {
			merged[key] += value
		}
	}
	return merged
}

type wordCount struct {
	word string

	count int
}

func formatTopK(counts map[string]int, topCount int) string {
	top := make([]wordCount, 0, topCount+1)
	for word, count := range counts {
		entry := wordCount{word: word, count: count}
		insertIntoTopK(&top, entry, topCount)
	}
	return renderTopK(top)
}

func insertIntoTopK(top *[]wordCount, entry wordCount, topCount int) {
	currentLength := len(*top)
	if currentLength < topCount {
		position := currentLength
		for position > 0 && beats(entry, (*top)[position-1]) {
			position--
		}
		*top = append(*top, wordCount{})
		copy((*top)[position+1:], (*top)[position:currentLength])
		(*top)[position] = entry
		return
	}
	if !beats(entry, (*top)[topCount-1]) {
		return
	}
	position := topCount - 1
	for position > 0 && beats(entry, (*top)[position-1]) {
		position--
	}
	copy((*top)[position+1:], (*top)[position:topCount-1])
	(*top)[position] = entry
}

func beats(left wordCount, right wordCount) bool {
	if left.count != right.count {
		return left.count > right.count
	}
	return left.word < right.word
}

func renderTopK(top []wordCount) string {
	buffer := make([]byte, 0, len(top)*32)
	for index := 0; index < len(top); index++ {
		if index > 0 {
			buffer = append(buffer, '\n')
		}
		entry := top[index]
		buffer = append(buffer, entry.word...)
		buffer = append(buffer, '\t')
		buffer = appendInt(buffer, entry.count)
	}
	return string(buffer)
}

func appendInt(out []byte, value int) []byte {
	if value == 0 {
		return append(out, '0')
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
	return append(out, digits[position:]...)
}
