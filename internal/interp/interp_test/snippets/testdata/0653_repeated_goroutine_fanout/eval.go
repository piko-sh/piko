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
			counts[string(chunk[tokenStart:index])]++
			tokenStart = -1
		}
	}
	if tokenStart >= 0 {
		counts[string(chunk[tokenStart:])]++
	}
	return counts
}

func parallelStep(chunks [][]byte) int {
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

func run() int {
	repeatedFragment := make([]byte, 0, 1024*44)
	for i := 0; i < 1024; i++ {
		fragment := []byte("the quick brown fox jumps over the lazy dog ")
		repeatedFragment = append(repeatedFragment, fragment...)
	}
	chunks := make([][]byte, 16)
	for i := 0; i < 16; i++ {
		chunks[i] = repeatedFragment
	}
	accumulator := 0
	for iteration := 0; iteration < 3; iteration++ {
		accumulator += parallelStep(chunks)
	}
	return accumulator
}
