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

func TestEvalChannels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "buffered channel send and receive",
			code: `ch := make(chan int, 1)
ch <- 42
v := <-ch
v`,
			expect: int64(42),
		},
		{
			name: "channel with goroutine",
			code: `ch := make(chan int, 1)
go func() {
	ch <- 99
}()
v := <-ch
v`,
			expect: int64(99),
		},
		{
			name: "select with default",
			code: `ch := make(chan int, 1)
ch <- 7
result := 0
select {
case v := <-ch:
	result = v
default:
	result = -1
}
result`,
			expect: int64(7),
		},
		{
			name: "select send case",
			code: `ch := make(chan int, 1)
result := 0
select {
case ch <- 42:
	result = 1
default:
	result = -1
}
result`,
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

func TestEvalChannelTyped(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "string channel",
			code: `ch := make(chan string, 1)
ch <- "hello"
v := <-ch
v`,
			expect: "hello",
		},
		{
			name: "float channel",
			code: `ch := make(chan float64, 1)
ch <- 3.14
v := <-ch
v`,
			expect: 3.14,
		},
		{
			name: "bool channel",
			code: `ch := make(chan bool, 1)
ch <- true
v := <-ch
v`,
			expect: true,
		},
		{
			name: "select recv string channel",
			code: `ch := make(chan string, 1)
ch <- "world"
result := ""
select {
case v := <-ch:
	result = v
default:
	result = "none"
}
result`,
			expect: "world",
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

func TestEvalChanRecvCommaOk(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{
			name: "recv ok true",
			code: `ch := make(chan int, 1)
ch <- 42
v, ok := <-ch
_ = v
ok`,
			expect: true,
		},
		{
			name: "recv ok value",
			code: `ch := make(chan int, 1)
ch <- 99
v, ok := <-ch
_ = ok
v`,
			expect: int64(99),
		},
		{
			name: "recv closed ok false",
			code: `ch := make(chan int)
close(ch)
_, ok := <-ch
ok`,
			expect: false,
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
