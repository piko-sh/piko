package main

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
)

func worker(wg *sync.WaitGroup, counter *atomic.Int64) {
	defer wg.Done()
	chunk := strings.Repeat("g", 100)
	localTotal := int64(0)
	for range 5000 {
		localTotal += int64(len(chunk))
		_ = chunk + "x"
	}
	counter.Add(localTotal)
}

func entrypoint() string {
	var wg sync.WaitGroup
	var totalLen atomic.Int64
	const workers = 5
	wg.Add(workers)
	for range workers {
		go worker(&wg, &totalLen)
	}
	wg.Wait()
	return fmt.Sprintf("workers=%d totalLen=%d", workers, totalLen.Load())
}

func main() {
	fmt.Println(entrypoint())
}
