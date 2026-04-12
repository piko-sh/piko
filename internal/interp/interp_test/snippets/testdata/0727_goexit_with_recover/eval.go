package main

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
)

func run() string {
	var wg sync.WaitGroup
	var defersRan []string
	var mu sync.Mutex

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			r := recover()
			mu.Lock()
			defersRan = append(defersRan, fmt.Sprintf("recover=%v", r))
			mu.Unlock()
		}()
		defer func() {
			mu.Lock()
			defersRan = append(defersRan, "inner-defer")
			mu.Unlock()
		}()
		runtime.Goexit()
		mu.Lock()
		defersRan = append(defersRan, "unreachable")
		mu.Unlock()
	}()
	wg.Wait()

	sort.Strings(defersRan)
	result := fmt.Sprintf("count=%d;", len(defersRan))
	for _, d := range defersRan {
		result += d + ";"
	}
	return result
}
