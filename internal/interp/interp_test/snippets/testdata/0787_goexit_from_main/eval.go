package main

import (
	"fmt"
	"sync"
)

var log []string
var logMu sync.Mutex

func appendLog(s string) {
	logMu.Lock()
	log = append(log, s)
	logMu.Unlock()
}

func run() string {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		appendLog("worker-ran")
	}()
	wg.Wait()
	appendLog("main-after-wait")

	result := fmt.Sprintf("count=%d;", len(log))
	for _, l := range log {
		result += l + ";"
	}
	return result
}
