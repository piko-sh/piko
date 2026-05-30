package main

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

func run() int {
	corpus := make([]byte, 0, 1024*44)
	for index := 0; index < 1024; index++ {
		fragment := []byte("the quick brown fox jumps over the lazy dog ")
		corpus = append(corpus, fragment...)
	}
	const N = 16
	chunks := make([][]byte, N)
	for index := 0; index < N; index++ {
		chunks[index] = corpus
	}
	results := make([]map[string]int, N)
	done := make(chan int, N)
	for index := 0; index < N; index++ {
		go func(slot int, chunk []byte) {
			results[slot] = tokenise(chunk)
			done <- slot
		}(index, chunks[index])
	}
	for completed := 0; completed < N; completed++ {
		<-done
	}
	total := 0
	for _, local := range results {
		total += local["the"]
	}
	return total
}
