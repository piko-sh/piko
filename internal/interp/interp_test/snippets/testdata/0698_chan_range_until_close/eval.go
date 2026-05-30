package main

import (
	"fmt"
	"sync"
)

func singleProducerClose() int {
	ch := make(chan int, 3)
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		close(ch)
	}()

	total := 0
	for value := range ch {
		total += value
	}
	return total
}

func multiProducerFanIn() int {
	ch := make(chan int, 8)
	var wg sync.WaitGroup

	for worker := 0; worker < 3; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < 4; i++ {
				ch <- workerID*10 + i
			}
		}(worker)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	count := 0
	for range ch {
		count++
	}
	return count
}

func drainAfterClose() string {
	ch := make(chan int, 3)
	ch <- 1
	ch <- 2
	ch <- 3
	close(ch)

	result := ""
	for v := range ch {
		result += fmt.Sprintf("%d,", v)
	}

	v, ok := <-ch
	result += fmt.Sprintf("post:v=%d/ok=%v", v, ok)
	return result
}

func run() string {
	out := ""
	out += fmt.Sprintf("singleProducer:%d;", singleProducerClose())
	out += fmt.Sprintf("multiProducer:%d;", multiProducerFanIn())
	out += "drain:" + drainAfterClose()
	return out
}
