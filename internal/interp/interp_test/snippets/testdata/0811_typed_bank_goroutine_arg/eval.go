package main

import "sync"

func run() int {
	chunks := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("gamma"),
		[]byte("delta"),
	}

	results := make([]int, len(chunks))
	var wg sync.WaitGroup
	wg.Add(len(chunks))
	for i, chunk := range chunks {
		go func(index int, body []byte) {
			defer wg.Done()
			results[index] = len(body)
		}(i, chunk)
	}
	wg.Wait()

	total := 0
	for _, value := range results {
		total += value
	}
	return total
}
