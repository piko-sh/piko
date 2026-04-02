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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalChannelRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "buffered_channel",
			code: `func sum() int {
	ch := make(chan int, 3)
	ch <- 10
	ch <- 20
	ch <- 30
	close(ch)
	total := 0
	for v := range ch {
		total += v
	}
	return total
}
sum()`,
			expect: int64(60),
		},
		{
			name: "closed_after_sends_via_goroutine",
			code: `func collect() int {
	ch := make(chan int, 5)
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		close(ch)
	}()
	sum := 0
	for v := range ch {
		sum += v
	}
	return sum
}
collect()`,
			expect: int64(15),
		},
		{
			name: "empty_channel",
			code: `func empty() int {
	ch := make(chan int)
	close(ch)
	count := 0
	for _ = range ch {
		count++
	}
	return count
}
empty()`,
			expect: int64(0),
		},
		{
			name: "string_channel",
			code: `func concat() string {
	ch := make(chan string, 2)
	ch <- "hello"
	ch <- " world"
	close(ch)
	result := ""
	for s := range ch {
		result += s
	}
	return result
}
concat()`,
			expect: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err, "code: %s", tt.code)
			require.Equal(t, tt.expect, result, "code: %s", tt.code)
		})
	}
}
