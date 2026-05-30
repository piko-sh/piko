package main

import "sync"

func tokenise(chunk []byte) map[string]int {
	counts := make(map[string]int)
	tokenStart := -1
	for index := 0; index < len(chunk); index++ {
		current := chunk[index]
		if (current >= 'a' && current <= 'z') || (current >= 'A' && current <= 'Z') {
			if tokenStart < 0 {
				tokenStart = index
			}
		} else if tokenStart >= 0 {
			emitToken(counts, chunk[tokenStart:index])
			tokenStart = -1
		}
	}
	if tokenStart >= 0 {
		emitToken(counts, chunk[tokenStart:])
	}
	return counts
}

func emitToken(counts map[string]int, word []byte) {
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

func run() int {
	vocabulary := []string{
		"the", "of", "and", "to", "a", "in", "that", "he", "his", "for",
		"it", "was", "with", "as", "I", "had", "not", "be", "is", "you",
		"have", "by", "this", "from", "or", "but", "an", "they", "which",
		"one", "would", "all", "their", "we", "she", "her", "if", "no",
		"when", "what", "who", "will", "said", "do", "are", "my", "your",
		"there", "more", "out", "him", "so", "up", "into", "than", "could",
		"some", "down", "very", "now", "any", "where", "then", "before",
		"after", "Edmond", "Dantes", "Count", "Monte", "Cristo", "Abbe",
	}
	repeatedFragment := make([]byte, 0, 32768*8)
	for i := 0; i < 32768; i++ {
		word := vocabulary[i%len(vocabulary)]
		for j := 0; j < len(word); j++ {
			repeatedFragment = append(repeatedFragment, word[j])
		}
		repeatedFragment = append(repeatedFragment, ' ')
	}
	corpus := make([]byte, 0, len(repeatedFragment)*16)
	for i := 0; i < 16; i++ {
		corpus = append(corpus, repeatedFragment...)
	}
	chunkSize := len(corpus) / 16
	chunks := make([][]byte, 16)
	for i := 0; i < 16; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == 15 {
			end = len(corpus)
		}
		chunks[i] = corpus[start:end]
	}
	results := make([]map[string]int, len(chunks))
	var waitGroup sync.WaitGroup
	for index := 0; index < len(chunks); index++ {
		waitGroup.Add(1)
		go func(slot int, chunk []byte) {
			defer waitGroup.Done()
			results[slot] = tokenise(chunk)
		}(index, chunks[index])
	}
	waitGroup.Wait()
	total := 0
	for _, local := range results {
		total += local["the"]
	}
	return total
}
