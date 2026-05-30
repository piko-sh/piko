package main

import (
	"fmt"
	"sort"
	"sync"
)

func collectViaClosure() []int {
	closures := make([]func() int, 0)
	for _, v := range []int{10, 20, 30, 40} {
		closures = append(closures, func() int { return v })
	}
	out := make([]int, len(closures))
	for index, fn := range closures {
		out[index] = fn()
	}
	return out
}

func collectViaPointer() []int {
	pointers := make([]*int, 0)
	for _, v := range []int{1, 2, 3} {
		pointers = append(pointers, &v)
	}
	out := make([]int, len(pointers))
	for index, p := range pointers {
		out[index] = *p
	}
	return out
}

func collectViaGoroutine() []int {
	var wg sync.WaitGroup
	collected := make([]int, 0, 4)
	var mu sync.Mutex
	for _, v := range []int{100, 200, 300, 400} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			collected = append(collected, v)
			mu.Unlock()
		}()
	}
	wg.Wait()
	sort.Ints(collected)
	return collected
}

func run() string {
	a := collectViaClosure()
	b := collectViaPointer()
	c := collectViaGoroutine()
	return fmt.Sprintf("closure:%v;pointer:%v;goroutine:%v", a, b, c)
}
