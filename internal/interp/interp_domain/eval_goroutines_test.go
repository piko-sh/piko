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

func TestEvalGoroutines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "goroutine sends on channel",
			code: `ch := make(chan int, 1)
go func() {
	ch <- 100
}()
v := <-ch
v`,
			expect: int64(100),
		},
		{
			name: "multiple goroutines sum",
			code: `ch := make(chan int, 3)
go func() { ch <- 10 }()
go func() { ch <- 20 }()
go func() { ch <- 30 }()
a := <-ch
b := <-ch
c := <-ch
a + b + c`,
			expect: int64(60),
		},
		{
			name: "goroutine with string argument",
			code: `ch := make(chan string, 1)
go func(message string) {
	ch <- message
}("hello")
v := <-ch
v`,
			expect: "hello",
		},
		{
			name: "goroutine with float argument",
			code: `ch := make(chan int, 1)
go func(x float64) {
	ch <- int(x * 2)
}(3.5)
v := <-ch
v`,
			expect: int64(7),
		},
		{
			name: "goroutine with bool argument",
			code: `ch := make(chan int, 1)
go func(flag bool) {
	if flag {
		ch <- 1
	} else {
		ch <- 0
	}
}(true)
v := <-ch
v`,
			expect: int64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
