// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package interp_domain

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalGoroutineFanOutSharedSliceWriteIsDeterministic(t *testing.T) {
	t.Parallel()

	const code = `func run() int {
	results := make([]int, 16)
	done := make(chan int, 16)
	for index := 0; index < 16; index++ {
		go func(slot int) {
			results[slot] = slot + 1
			done <- slot
		}(index)
	}
	for completed := 0; completed < 16; completed++ {
		<-done
	}
	total := 0
	for _, value := range results {
		total += value
	}
	return total
}
run()`

	const expected = int64(1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 + 10 + 11 + 12 + 13 + 14 + 15 + 16)

	for iteration := range 64 {
		service := NewService()
		result, err := service.Eval(context.Background(), code)
		require.NoErrorf(t, err, "iteration %d", iteration)
		require.Equalf(t, expected, result, "iteration %d", iteration)
	}
}

func TestEvalGoroutineFanOutByteSliceParamIsDeterministic(t *testing.T) {
	t.Parallel()

	const code = `func run() int {
	corpus := make([]byte, 100)
	for index := 0; index < 100; index++ {
		corpus[index] = byte('a')
	}
	chunks := make([][]byte, 16)
	for index := 0; index < 16; index++ {
		chunks[index] = corpus
	}
	lengths := make([]int, 16)
	done := make(chan int, 16)
	for index := 0; index < 16; index++ {
		go func(slot int, chunk []byte) {
			lengths[slot] = len(chunk)
			done <- slot
		}(index, chunks[index])
	}
	for completed := 0; completed < 16; completed++ {
		<-done
	}
	total := 0
	for _, value := range lengths {
		total += value
	}
	return total
}
run()`

	const expected = int64(16 * 100)

	for iteration := range 64 {
		service := NewService()
		result, err := service.Eval(context.Background(), code)
		require.NoErrorf(t, err, "iteration %d", iteration)
		require.Equalf(t, expected, result, "iteration %d", iteration)
	}
}

func TestEvalRepeatedGoroutineFanOutIsDeterministic(t *testing.T) {
	t.Parallel()

	const code = `func parallelStep(values []int) int {
	results := make([]int, len(values))
	done := make(chan int, len(values))
	for index := 0; index < len(values); index++ {
		go func(slot int, value int) {
			results[slot] = value * 2
			done <- slot
		}(index, values[index])
	}
	for completed := 0; completed < len(values); completed++ {
		<-done
	}
	total := 0
	for _, value := range results {
		total += value
	}
	return total
}
func run() int {
	values := make([]int, 16)
	for index := 0; index < 16; index++ {
		values[index] = index + 1
	}
	accumulator := 0
	for iteration := 0; iteration < 5; iteration++ {
		accumulator += parallelStep(values)
	}
	return accumulator
}
run()`

	const expected = int64(5 * 2 * (1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 + 10 + 11 + 12 + 13 + 14 + 15 + 16))

	for iteration := range 32 {
		service := NewService()
		result, err := service.Eval(context.Background(), code)
		require.NoErrorf(t, err, "iteration %d", iteration)
		require.Equalf(t, expected, result, "iteration %d", iteration)
	}
}

func TestEvalConcurrentEvaluationsDoNotInterfere(t *testing.T) {
	t.Parallel()
	service := NewService()
	const workers = 8
	const iterationsPerWorker = 16
	const code = `func run() int {
	values := make([]int, 4)
	values[0] = 1
	values[1] = 2
	values[2] = 3
	values[3] = 4
	doubleAll := func() {
		for index := 0; index < len(values); index++ {
			values[index] = values[index] * 2
		}
	}
	doubleAll()
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}
run()`
	const expected = int64(2 + 4 + 6 + 8)

	var wg sync.WaitGroup
	wg.Add(workers)
	errors := make(chan error, workers*iterationsPerWorker)
	for workerIndex := range workers {
		go func(worker int) {
			defer wg.Done()
			for iteration := range iterationsPerWorker {
				result, err := service.Eval(context.Background(), code)
				if err != nil {
					errors <- err
					continue
				}
				if result != expected {
					errors <- &concurrentEvalMismatch{worker: worker, iteration: iteration, expected: expected, got: result}
				}
			}
		}(workerIndex)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

type concurrentEvalMismatch struct {
	got       any
	expected  any
	worker    int
	iteration int
}

func (e *concurrentEvalMismatch) Error() string {
	return fmt.Sprintf("worker %d iteration %d: expected %v, got %v",
		e.worker, e.iteration, e.expected, e.got)
}
