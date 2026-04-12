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
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalGoroutineMapStringKeysSurviveArenaReset(t *testing.T) {
	t.Parallel()
	previousGoroutines := runtime.GOMAXPROCS(1)
	t.Cleanup(func() {
		runtime.GOMAXPROCS(previousGoroutines)
	})

	const code = `func tokenise(chunk []byte) map[string]int {
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
	chunks := make([][]byte, 16)
	for index := 0; index < 16; index++ {
		chunks[index] = corpus
	}
	results := make([]map[string]int, 16)
	done := make(chan int, 16)
	for index := 0; index < 16; index++ {
		go func(slot int, chunk []byte) {
			results[slot] = tokenise(chunk)
			done <- slot
		}(index, chunks[index])
	}
	for completed := 0; completed < 16; completed++ {
		<-done
	}
	total := 0
	for _, local := range results {
		total += local["the"]
	}
	return total
}
run()`

	const expected = int64(16 * 2048)

	for iteration := range 8 {
		service := NewService()
		result, err := service.Eval(context.Background(), code)
		require.NoErrorf(t, err, "iteration %d", iteration)
		require.Equalf(t, expected, result, "iteration %d", iteration)
	}
}
